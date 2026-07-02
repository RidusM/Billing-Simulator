package config

import "time"

type (
	Config struct {
		App      App      `env-prefix:"APP_"`
		Database Database `env-prefix:"DB_"`
		Cache    Cache    `env-prefix:"CACHE_"`
		Kafka    Kafka    `env-prefix:"KAFKA_"`
		HTTP     HTTP     `env-prefix:"HTTP_"`
		Logger   Logger   `env-prefix:"LOGGER_"`
		Env      string   `env:"ENV" env-default:"local" validate:"oneof=local dev prod"`
	}

	App struct {
		Name    string `env:"NAME" env-default:"billing-stripe-simulator" validate:"required"`
		Version string `env:"VERSION" env-default:"1.0.0" validate:"required"`
	}

	Database struct {
		Host     string `env:"HOST"     env-default:"localhost" validate:"required"`
		Port     string `env:"PORT"     env-default:"5432"      validate:"required"`
		User     string `env:"USER"     env-default:"postgres"  validate:"required"`
		Password string `env:"PASSWORD" env-default:"postgres"  validate:"required"`
		Name     string `env:"NAME"     env-default:"billing"   validate:"required"`
		SSLMode  string `env:"SSL_MODE" env-default:"disable"   validate:"required"`

		MaxPoolSize    int32         `env:"MAX_POOL_SIZE"    env-default:"100"`
		ConnAttempts   int           `env:"CONN_ATTEMPTS"    env-default:"5"     validate:"min=1,max=10"`
		BaseRetryDelay time.Duration `env:"BASE_RETRY_DELAY" env-default:"100ms" validate:"gte=10ms,lte=10s"`
		MaxRetryDelay  time.Duration `env:"MAX_RETRY_DELAY"  env-default:"5s"    validate:"gte=100ms,lte=30s,gtefield=BaseRetryDelay"`
	}

	Cache struct {
		Host     string `env:"HOST"           env-default:"localhost" validate:"required"`
		Port     string `env:"PORT"           env-default:"6379"      validate:"gte=1,lte=65535"`
		Password string `env:"PASSWORD"       env-default:""`
		DB       int    `env:"DB"             env-default:"0"`

		PoolTimeout  time.Duration `env:"POOL_TIMEOUT"   env-default:"3s"        validate:"gte=1s,lte=30s"`
		PoolSize     int           `env:"POOL_SIZE"      env-default:"20"        validate:"min=1,max=100"`
		MinIdleConns int           `env:"MIN_IDLE_CONNS" env-default:"10"        validate:"min=1,max=100"`
		TTL          time.Duration `env:"TTL"            env-default:"1h"`
	}

	Kafka struct {
		Brokers           []string `env:"BROKERS"            env-separator:"," env-default:"localhost:9092"`
		InvoiceTopic      string   `env:"INVOICE_TOPIC"      env-default:"billing.invoices"`
		SubscriptionTopic string   `env:"SUBSCRIPTION_TOPIC" env-default:"billing.subscriptions"`
		DLQTopic          string   `env:"DLQ_TOPIC"          env-default:"billing.dlq"`
		ConsumerGroup     string   `env:"CONSUMER_GROUP"     env-default:"billing-service"`

		MaxAttempts    int           `env:"MAX_ATTEMPTS"       env-default:"3"`
		BaseRetryDelay time.Duration `env:"BASE_RETRY_DELAY"   env-default:"10ms"`
		MaxRetryDelay  time.Duration `env:"MAX_RETRY_DELAY"    env-default:"100ms"`
	}

	HTTP struct {
		Host              string        `env:"HOST"                env-default:"0.0.0.0" validate:"required"`
		Port              string        `env:"PORT"                env-default:"8080"    validate:"required"`
		ReadTimeout       time.Duration `env:"READ_TIMEOUT"        env-default:"5s"      validate:"gte=1s,lte=30s"`
		WriteTimeout      time.Duration `env:"WRITE_TIMEOUT"       env-default:"5s"      validate:"gte=1s,lte=30s"`
		IdleTimeout       time.Duration `env:"IDLE_TIMEOUT"        env-default:"60s"     validate:"gte=1s,lte=300s"`
		ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT"    env-default:"10s"     validate:"gte=1s,lte=30s"`
		ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" env-default:"5s"      validate:"gte=1s,lte=30s"`
		MaxHeaderBytes    int           `env:"MAX_HEADER_BYTES"    env-default:"1048576" validate:"required,gte=1024,lte=10485760"`
	}

	Logger struct {
		Level    string `env:"LEVEL"       env-default:"info"                        validate:"oneof=debug info warn error"`
		Filename string `env:"FILENAME"    env-default:"./logs/billing-service.log"`

		MaxSize    int `env:"MAX_SIZE"    env-default:"100"                         validate:"min=1,max=1000"`
		MaxBackups int `env:"MAX_BACKUPS" env-default:"3"                           validate:"min=0,max=20"`
		MaxAge     int `env:"MAX_AGE"     env-default:"28"                          validate:"min=1,max=365"`
	}
)
