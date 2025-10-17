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
```

Put that file in one of these locations:

- ./gopenbridge.yaml
- ~/.gopenbridge.yaml
- ~/.config/gopenbridge/config.yaml


Run the binary:

```
./gopenbridge
```

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
