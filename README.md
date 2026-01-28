# eventing-rate-limiter
Rate limiter built from the Couchbase Eventing service. 
This respository is part of the blog post "Build a Rate Limiter With Couchbase Eventing".

## File Description

|        File Name       |                                 Description                                 |                                                                                                              Notes                                                                                                             |
|:----------------------:|:---------------------------------------------------------------------------:|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------:|
| `server-api-spec.yaml` | The OpenAPI specification of the server API endpoints                       | To view a more interactive version of the above OpenAPI specification, simply copy and paste the specification into the official online [Swagger editor](https://editor.swagger.io/).                                          |
|  `event-generator.go`  | Go Program to Generate Events to Trigger the Rate-Limiter Eventing Function | The source code of the Go program allows you to select how many requests each user sends, i.e., below their rate limit, at their rate limit or above their rate limit, by changing the value of the constant named `whatToDo`. |
| `eventing-function.js` | Rate-Limiter Eventing Function                                              |                                                                                                                -                                                                                                               |
|    `user-loader.go`    | Go Program to Load 100 Users into the Couchbase Cluster                     |                                                                                                                -                                                                                                               |
|       `server.go`      | Go Program that Hosts the External REST API Endpoints                       | A `GET` request to the `/my-llm` endpoint will return a JSON value containing a `counter` field whose value indicates the number of `POST` requests the endpoint received.                                                     |

## Go Workspace and Builds

This repo is a Go workspace with multiple mini-projects. Use the workspace at the root to work across the Go modules together.

### Setup order

1. Deploy the Eventing function in Couchbase Server using the blog post steps.
1. Run the server, user loader, and event generator.

### Configuration

The bucket, scope, and collection names are intentionally kept in code to match the blog post. The following values can be overridden via env vars:

- `CB_CONNECTION_STRING` (default: `localhost:12000`)
- `CB_USERNAME` (default: `Administrator`)
- `CB_PASSWORD` (default: `asdasd`)
- `USERS_JSON_PATH` (default: `/Users/rishitchaudhary/Dev/roughpad/users.json`)

### Workspace usage

From the repo root:

```
go work sync
```

### Build and run

Event generator:

```
go build -C event-generator
go run -C event-generator .
```

User loader:

```
go build -C user-loader
go run -C user-loader .
```

Server:

```
go build -C server
go run -C server .
```

To stop the running server, press `ctrl` + `c` in the terminal where it is running.

## Helpful Commands

1. To clear all the documents in the `my-llm.users.events` keyspace, run the SQL++ query: ``delete from `my-llm`.users.events;``
1. To clear all the documents in the `rate-limiter.my-llm.tracker` keyspace, run the SQL++ query: ``delete from `rate-limiter`.`my-llm`.tracker;``

