variable "slack_bot_token" {
  description = "the one starts with xoxb"
}
variable "slack_signing_secret" {
  description = "Slack signs the requests we send you using this secret. Confirm that each request comes from Slack by verifying its unique signature."
}

variable "commit_sha" {
  description = "the commit short sha"
}
