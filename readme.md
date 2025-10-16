# gopenbridge

Shamelessly copied from `fakerybakery/openbridge` and rewrote it in go for performance. I just wanted to use claude code with GroqCloud.

# Build

```
make build
```

Tested with claude code version `v2.0.19`.

Working the `claude` command:
```
ANTHROPIC_DEFAULT_HAIKU_MODEL=openai/gpt-oss-120b ANTHROPIC_DEFAULT_SONNET_MODEL=moonshotai/kimi-k2-instruct-0905  ANTHROPIC_BASE_URL=http://0.0.0.0:8323 claude
```
