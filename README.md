# Development

## Launching Local Development Server

launch development server with:

```
docker-compose up -d
```


## create tunnel for slack


```
 nix shell nixpkgs#cloudflared -c cloudflared tunnel --url http://localhost:3000  
```





navigate to your [app management
dashboard](https://api.slack.com/apps) on slack, and slack the bot
application that is being developed.

under the *Features* sidebar, there is a menu option called `Slash
Commands`. Select that menu option for a list of Slash commands for
that current bot. 

Edit the command that is being developed to use the ngrok url, from
our screenshot it is: `https://3475afdd4195.ngrok.io`. Note that our
web server serves slash commands under the path `/slash`.


![slack slash command menu](./assets/edit-command.png)

save your changes and test your command in slack

## rebuild webserver

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