# stripe2ntfy

A lightweight Stripe webhook server that forwards events to [ntfy](https://github.com/binwiederhier/ntfy).

## Usage

1. **Build**

```bash
go build github.com/vytskalt/stripe2ntfy/cmd/stripe2ntfy
```

2. **Run**

```bash
STRIPE_WEBHOOK_SECRET=whsec_... NTFY_URL=https://ntfy.sh/your-topic ./stripe2ntfy
```

The server will start and listen on port `127.0.0.1:3000`.

## Configuration

The following environment variables are available.

| Variable                | Required | Description                                                                                                   |
|-------------------------|----------|---------------------------------------------------------------------------------------------------------------|
| `STRIPE_WEBHOOK_SECRET` | Yes      | The webhook signing secret from your Stripe dashboard.                                                        |
| `NTFY_URL`              | Yes      | The URL of the Ntfy topic to which notifications will be sent (e.g., `https://ntfy.sh/mytopic`).              |
| `NTFY_TOKEN`            | No       | A bearer token for Ntfy authentication. If provided, this will be used for `Bearer` token authentication.     |
| `NTFY_USERNAME`         | No       | The username for Ntfy basic authentication. `NTFY_PASSWORD` must also be set. Ignored if `NTFY_TOKEN` is set. |
| `NTFY_PASSWORD`         | No       | The password for Ntfy basic authentication. `NTFY_USERNAME` must also be set. Ignored if `NTFY_TOKEN` is set. |
| `LISTEN_ADDR`           | No       | The IP and port to make the server listen on, defaults to `127.0.0.1:3000`.                                   |

### Authentication ([docs](https://docs.ntfy.sh/publish/?h=bearer#authentication))

* **Bearer Token Authentication:** Set the `NTFY_TOKEN` environment variable.
* **Basic Authentication:** Set both the `NTFY_USERNAME` and `NTFY_PASSWORD` environment variables.

If `NTFY_TOKEN` is provided, it will be used for authentication, and basic auth credentials will be ignored. If no authentication variables are set, the request will be sent to Ntfy without an `Authorization` header.