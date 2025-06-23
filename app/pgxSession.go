package app

import (
	"context"
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// PGStore represents the currently configured session store.
type PGStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options
	Path    string
	Conn    *pgxpool.Pool
}

// PGSession type
type PGSession struct {
	ID         string
	Key        string
	Data       string
	CreatedOn  time.Time
	ModifiedOn time.Time
	ExpiresOn  time.Time
}

// NewPGStoreFromConn creates a new PGStore instance from an existing
// database connection.
// This will also create the database schema needed by pgstore.
func NewPGStoreFromPool(pool *pgxpool.Pool, keyPairs ...[]byte) (*PGStore, error) {
	dbStore := &PGStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 30,
			HttpOnly: false,
			Secure:   false,
		},
		Conn: pool,
	}

	return dbStore, nil
}

// Get Fetches a session for a given name after it has been added to the
// registry.
func (db *PGStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(db, name)
}

// New returns a new session for the given name without adding it to the registry.
func (db *PGStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(db, name)
	if session == nil {
		return nil, nil
	}

	opts := *db.Options
	session.Options = &(opts)
	session.IsNew = true

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, db.Codecs...)
		if err == nil {
			err = db.load(session)
			if err == nil {
				session.IsNew = false
			} else if errors.Cause(err) == pgx.ErrNoRows {
				err = nil
			}
		}
	}

	db.MaxAge(db.Options.MaxAge)

	return session, err
}

// Save saves the given session into the database and deletes cookies if needed
func (db *PGStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Set delete if max-age is < 0
	if session.Options.MaxAge < 0 {
		if err := db.destroy(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		// Generate a random session ID key suitable for storage in the DB
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32),
			), "=")
	}

	if err := db.save(session); err != nil {
		return err
	}

	// Keep the session ID key in a cookie so it can be looked up in DB later.
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, db.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new PGStore is 4096. PostgreSQL allows for max
// value sizes of up to 1GB (http://www.postgresql.org/docs/current/interactive/datatype-character.html)
func (db *PGStore) MaxLength(l int) {
	for _, c := range db.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (db *PGStore) MaxAge(age int) {
	db.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range db.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// load fetches a session by ID from the database and decodes its content
// into session.Values.
func (db *PGStore) load(session *sessions.Session) error {
	var s PGSession

	err := db.selectOne(&s, session.ID)
	if err != nil {
		return err
	}

	return securecookie.DecodeMulti(session.Name(), string(s.Data), &session.Values, db.Codecs...)
}

// save writes encoded session.Values to a database record.
// writes to cliente_session_store table by default.
func (db *PGStore) save(session *sessions.Session) error {
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, db.Codecs...)
	if err != nil {
		return err
	}

	crOn := session.Values["created_at"]
	exOn := session.Values["expires_at"]

	var expiresOn time.Time

	createdOn, ok := crOn.(time.Time)
	if !ok {
		createdOn = time.Now()
	}

	if exOn == nil {
		expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
	} else {
		expiresOn = exOn.(time.Time)
		if expiresOn.Sub(time.Now().Add(time.Second*time.Duration(session.Options.MaxAge))) < 0 {
			expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
		}
	}

	s := PGSession{
		Key:        session.ID,
		Data:       encoded,
		CreatedOn:  createdOn,
		ExpiresOn:  expiresOn,
		ModifiedOn: time.Now(),
	}

	if session.IsNew {
		return db.insert(&s)
	}

	return db.update(&s)
}

// Delete session
func (db *PGStore) destroy(session *sessions.Session) error {
	_, err := db.Conn.Exec(context.Background(), "DELETE FROM publico.cliente_session_store WHERE key = $1", session.ID)
	return err
}

func (db *PGStore) selectOne(s *PGSession, key string) error {
	stmt := "SELECT id, key, data, created_at, modified_at, expires_at FROM publico.cliente_session_store WHERE key = $1"
	err := db.Conn.QueryRow(context.Background(), stmt, key).Scan(&s.ID, &s.Key, &s.Data, &s.CreatedOn, &s.ModifiedOn, &s.ExpiresOn)
	if err != nil {
		return errors.Wrapf(err, "Unable to find session in the database")
	}

	return nil
}

func (db *PGStore) insert(s *PGSession) error {
	stmt := `INSERT INTO publico.cliente_session_store (id, key, data, created_at, modified_at, expires_at)
           VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)`
	_, err := db.Conn.Exec(context.Background(), stmt, s.Key, s.Data, s.CreatedOn, s.ModifiedOn, s.ExpiresOn)

	return err
}

func (db *PGStore) update(s *PGSession) error {
	stmt := `UPDATE publico.cliente_session_store SET data=$1, modified_at=$2, expires_at=$3 WHERE key=$4`
	_, err := db.Conn.Exec(context.Background(), stmt, s.Data, s.ModifiedOn, s.ExpiresOn, s.Key)

	return err
}
