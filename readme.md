# gopenbridge

Shamelessly copied from `fakerybakery/openbridge` and rewrote it in go for performance. I just wanted to use claude code with GroqCloud.

# Build

```sh
make build
```

# Usage

Create config file:

For example:

./gopenbridge.yaml
```yaml
api_key: gsk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
base_url: https://api.groq.com/openai/v1
model: moonshotai/kimi-k2-instruct-0905
max_tokens: 14000
debug: true    # optional: enable verbose debug logging
```

Put that file in one of these locations:

- ./gopenbridge.yaml
- ~/.gopenbridge.yaml
- ~/.config/gopenbridge/config.yaml

### Using a Custom Config File Path

**Note**: gopenbridge does not currently support specifying a custom config file path via command-line arguments. The application only searches in the standard locations listed above.

If you need to use a config file from a non-standard path, use environment variables instead:

```bash
export OPENAI_API_KEY="gsk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
export OPENAI_BASE_URL="https://api.groq.com/openai/v1"
export OPENAI_MODEL="moonshotai/kimi-k2-instruct-0905"
export MAX_OUTPUT_TOKENS="14000"
export DEBUG="true"

./gopenbridge --host 0.0.0.0 --port 8323
```


Run the binary:

```
./gopenbridge
```
To enable debug logging, set environment variable `DEBUG=true` or add `debug: true` in your config file.

Install `claude-code`

```sh
npm install -g @anthropic-ai/claude-code@2.0.19
```

Not sure if other versions will work.

Use this `claude` command:
```
ANTHROPIC_DEFAULT_HAIKU_MODEL=openai/gpt-oss-120b \
    ANTHROPIC_DEFAULT_SONNET_MODEL=moonshotai/kimi-k2-instruct-0905 \
    ANTHROPIC_BASE_URL=http://0.0.0.0:8323 claude
```
