# README #

### this project contains the next tasks ###
* endpoints for micuenta

### you need to install this packages using go ###
* go install github.com/githubnemo/CompileDaemon      # autoreload app on change
* go install github.com/swaggo/swag/cmd/swag@latest   # install in the OS swag command
* go get -u github.com/gin-gonic/gin                  # framework
* go get -u github.com/joho/godotenv                  # cargar variables de .env
* go get -u github.com/gin-contrib/i18n               # internacionalizacion de msjs
* go get -u github.com/go-sql-driver/mysql            # mysql driver
* go get -u github.com/jackc/pgx/v5                   # postgresql driver
* go get -u github.com/jackc/pgx/v5/pgxpool           # postgresql driver
* go get -u github.com/go-playground/validator/v10    # validadores forms
* go get -u gopkg.in/natefinch/lumberjack.v2          # logrotate
* go get -u github.com/go-co-op/gocron/v2             # crons
* go get -u github.com/swaggo/gin-swagger             # library to handle documentation on the project
* go get -u github.com/swaggo/files                   # library to handle  documentation on the project
* go get -u github.com/gin-contrib/sessions           # use for handling sessions
* go get -u github.com/golang-jwt/jwt/v5              # use for jwt tokens
* go get -u golang.org/x/crypto                       # use for cryptography
* go get -u github.com/wagslane/go-password-validator # password strenght validator
* go get -u gopkg.in/gomail.v2                        # send email via smtp

### you need also to create a .env file below are the related vars ### 

```
  PORT=7003
  
  # use release or debug mode, in debug: all request are logged with the header and body
  GIN_MODE=debug

  # mysql call_center variables
  DB_MYSQL=user:password|@tcp(ip_address:port)/database_name
  MYSQL_MAX_CONN=5
  MYSQL_MIN_CONN=2

  # postgres variables
  DB_POSTGRES=postgres://user:password@ip_address:port/database_name
  PGSQL_MAX_CONN=5
  PGSQL_MIN_CONN=1

  # variables de sesion, max age is in seconds
  DOMAIN=""
  SECURE_COOKIE=false
  SECRET="iNbTj#CUI0h[=nHx;L}D>w=j[XOkt8|)|sda2nyYR6=ube}\Jt/K22^8q1T0@rsO"
  AUTH_MAX_AGE=600
  REFRESH_MAX_AGE=86400
  REFRESH_MAX_AGE_REMEMBER=2592000

  # Variables to use in Cors
  CORS_ORIGINS="https://domain.com,http://domain2.com"
  CORS_ALLOW_HEADERS="Content-Type, Content-Length, Accept, Accept-Language, Accept-Encoding, Origin, X-Access-Token, User-Agent"
  CORS_METHODS="GET,POST,DELETE"
  CORS_MAX_AGE="86400"

  # Variables to handle basic auth for access to api documentation url is /docs/index.html
  DOC_USER=username_here
  DOC_PASSWD=password_here

  # link for change password and max age of token in seconds
  LINK_CHANGE_PASSWORD="http://127.0.0.1/change-password?uuid="
  LINK_MAX_AGE=43200
  LINK_SECRET="asjas87as0askañs9a8s126126///%$%&HJKAJG&LKJ·$%%"

  # variables to handle smtp options
  SMTP_SERVER="mail.bessersolutions.com"
  SMTP_PORT="587"
  SMTP_EMAIL="norespuesta@bessersolutions.com"
  SMTP_PASSWORD="Besser89**"

  # variables to handle file uploads
  PAYMENT_UPLOAD_FOLDER="./public/uploads/payments"

```

### Example of job definition: in .crontab ###
#### must create .crontab file on root folder of project to operate cron jobs, checkout crontab_example.json ####
```
 .---------------- minute (0 - 59)
 |  .------------- hour (0 - 23)
 |  |  .---------- day of month (1 - 31)
 |  |  |  .------- month (1 - 12) OR jan,feb,mar,apr ...
 |  |  |  |  .---- day of week (0 - 6) (Sunday=0 or 7) OR sun,mon,tue,wed,thu,fri,sat
 |  |  |  |  |
 *  *  *  *  * 
```