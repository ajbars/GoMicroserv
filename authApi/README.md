## Auth Serive

It is responsible for JWT generation and verification by using private key.
One of the way to generate a key pair:
`ssh-keygen -t ecdsa -m PEM`

### Configuration
- env var `PRIVATE_KEY_PATH` path to private key file
- env var `REDIS_HOST` is a pair of host:port (without scheme)
