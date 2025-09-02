# Development

## Launching Local Development Server

launch development server with:

```
docker-compose up -d
```

## create tunnel for slack

```
nix shell nixpkgs#cloudflared -c cloudflared tunnel --url http://localhost:3333
```

## Rebuild webserver

```
docker-compose up --build -d app
```

## webserver logs

```
docker-compose logs -tf app
```

## restart ngrok
```shell
docker-compose restart ngrok
```

# k8s deployment

```
yes y | flux bootstrap git \
--url=ssh://git@github.com/r33drichards/lucksacks.git \
--branch=main \
--path=charts \
--namespace=default \
--private-key-file=/Users/robertwendt/.ssh/id_ed25519
```

## create secret from .env file
```sh
kubectl delete secret lucksacks-secret
kubectl create secret generic lucksacks-secret --from-env-file=.env
```

# Tailscale IP Address Detection Feature

This feature adds the ability to detect IP addresses mentioned in Slack messages and match them against devices on your Tailscale tailnet. When an IP address mentioned in a message matches a device on your tailnet, the bot will reply in the thread with the device name.

## Setup

1. Add the following environment variables to your `.env` file:

```
TAILSCALE_API_KEY=your_api_key
TAILSCALE_TAILNET_ID=your_tailnet_id
```

You can obtain your Tailscale API key from the [Tailscale admin console](https://login.tailscale.com/admin/settings/keys).
Your tailnet ID is usually your Tailscale account name, which can be found in the URL of your Tailscale admin console: `https://login.tailscale.com/admin/machines?tailnet=YOUR_TAILNET_ID`.

## How It Works

1. When a message is received in a channel, the bot scans the message for IP addresses using regular expressions.
2. For each detected IP address, the bot checks if it matches an IP address of a device on your tailnet.
3. If a match is found, the bot replies in the thread with a message indicating which device the IP address belongs to.

## Example

User: "Can someone check if 100.100.100.100 is online?"
Bot: "IP address 100.100.100.100 belongs to device `office-laptop` on your tailnet."