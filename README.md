# Slack Simple Private Message API

Take a simple incoming HTTP request and posts it to a Slack private message. This is complicated because you need to:

1. List all the users in the workspace to find the user ID with the given display_name.
2. Post to the user ID.

This is currently an App Engine application because I'm running it in their free tier.


To use it: Deploy this to App Engine, then send an HTTP POST with the following body:

```json
{
  "token": "Slack bot OAuth token",
  "display_name": "The display_name of the Slack user we want to message",
  "text": "The slack message text to send in a private message",
}
```

The service will perform the user list then proxy the message, returning the ok and error fields from the post message status.