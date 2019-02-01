# RSSH

Reverse SSH gateway.

## Configuration

See `.rssh.yml`

### Environment variables

You can use override any default and configuration sourced variable with the environment.
All environment variables are prefixed with `RSSH_`, and their name is constructed by taking
the capitalized dot separated path of your variable in `.rssh.yml`.
(e.g: `gatekeeper.ssh_port_range` => `RSSH_GATEKEEPER_SSH_PORT_RANGE`)

