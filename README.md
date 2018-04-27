__🚧 The Core Agent API is under active development. If you use build instrumentation on top of the Core Agent API, watch this repo to stay informed of potentially breaking changes. If you have questions, email support@scoutapp.com.__

# Scout Core Agent API

Using [Scout](https://scoutapp.com), but have an app in a non-supported language?

Well, if you can send messages over a Unix Domain Socket (hint: you can in every common language), you can add tracing and instrumentation to your app. The Scout Core Agent is designed to provide cross-language common functionality, and act as a backend to instrumentation in many languages.  It is implemented as a standalone binary.

There's no need to create a wrapper for a C SDK or write complex buffering, background-threaded logic. Besides ease-of-use, UDP sockets are an [efficient way to communicate](https://stackoverflow.com/a/29436429/1234395). The Core Agent is written in Rust, a naturally [fast language](http://benchmarksgame.alioth.debian.org/u64q/compare.php?lang=rust&lang2=node).

This repository contains documentation and examples for the Core Agent API. Note that the Core Agent itself is closed source.

## How the Core Agent works

About 80% of the logic required for an Application Performance Agent (APM) lies in language-agnostic code. The remaining 20% is the language instrumentation. Starting with Scout's Python monitoring agent, the language instrumentation communicates with the core agent, sending information like the start and stop of requests to the Core Agent. The core agent handles the aggregation of performance metrics and reporting data to Scout's servers.

The core agent is designed to run on the same host as the instrumented app.

## Quick Start

Anxious to see some example code? Checkout the [NodeJS example](https://github.com/scoutapp/core-agent-api/blob/master/examples/nodejs/app.js).

## Downloading the Core Agent

The binary is available at
`https://s3-us-west-1.amazonaws.com/scout-public-downloads/apm_core_agent/release/scout_apm_core-latest-x86_64-unknown-linux-gnu.tgz`,
with the name of the CPU and OS replaced by the build. `latest` always points at the newest release,
and exact versions are available too: `v1.1.2`

### Builds

* i686-unknown-linux-gnu
* x86_64-apple-darwin
* x86_64-unknown-linux-gnu

## Launching

The Core Agent binary has several modes it can launch in.

* `start` launches a server, to listen to incoming requests from a Language Agent.
* `probe` requests information from a running Core Agent
* `shutdown` requests that a running Core Agent shutdown

### Start

Launches the Core Agent, and starts listening for connections from a Language Agent.

```
$ core-agent start --socket /tmp/core-agent.sock
[2018-04-16T17:59:12][core_agent][INFO] Initializing logger with log level: Info
[2018-04-16T17:59:12][core_agent][INFO] Starting ScoutApm CoreAgent version: "1.0.1"
[2018-04-16T17:59:12][socket::server][INFO] Socket Server bound to socket at "/tmp/scout-agent.sock"
```

### Probe

Connects to a socket, and requests information from a running Core Agent

```
$ core-agent probe --socket /tmp/core-agent.sock
Agent found: CoreAgentVersion { raw: "1.0.1" }
```

### Shutdown

Connects to a socket, and shuts down a running Core Agent

#### Command Line Options

There are a number of options that can be passed to Core Agent when starting. They apply to all
launch modes.


| Flag            | Meaning  | Default |
|-----------------|----------|---------|
| `--log-level`   | `trace`, `debug`, `info`, `warn`, `error` | `info` |
| `--log-file`    | Log file | outputs to `stdout` |
| `--socket`      | What filesystem location to open a listening Unix Domain Socket | `$AGENT_DIR/core-agent.sock`
| `--config-file` | Specify a path to a toml formatted configuration file. Used only for internal debugging currently | None |

## Communications

### Socket

All communications in version 1.x are sent over a Unix Domain Socket. It's expected that other
adapters will exist in the future (http, grpc, etc).

#### Framing

Each message over the Socket is sent with a 4 byte big-endian length, then the bytes for the
message.

### Data Encoding

The data is current encoded as JSON. See the sections below for examples.

## Flow

* Core Agent is launched
* Language Agent opens one or more sockets
* Language Agent registers itself with the Org Key and Application name it is running
* Language Agent sends a series of Commands (Start & Stop requests, Start & Stop spans, Tag requests and spans)

If for any reason the socket closes, simply reopen it and re-register.

## Unregistered Commands

These may be sent only while a client is not yet registered.

### Register

```
{Register:  {
              app: String,
              key: String,
              api_version: String,
            }
}
```

### CoreAgentVersion

This also has a registered client equivalent.

Request the current version info of the Core Agent running. See Responses for more details of the
returned value.

```
{CoreAgentVersion: { }}
```

### CoreAgentShutdown

This also has a registered client equivalent.

Request that the Core Agent shuts down.

```
{CoreAgentShutdown: { }}
```

## Post-Registration Commands

These commands are permitted after a client is registered.

### StartRequest

Start a new Request. All Spans will reference the RequestId you send. RequestId needs to be a
globally unique string, but otherwise holds no information. A UUID4 is sufficient.

```
{StartRequest:  {
                  request_id: RequestId,
                  timestamp: Option<EventTimestamp>,
                }
}
```

### StartSpan

Start a new Span, belonging to a specific Request that was previously started. Optionally
(commonly), a span will be the child of another Span that occurred during this request.

For instance, a common pattern in Django is to wrap the "View" code in a span.

For example, each nested layer refers to its parent:

```
Start Span "View/search.index"
  Start Span "SQL/Data"
  Stop Span "SQL/Data"

  Start Span "Template/Render/search.html"
    Start Span "Template/Render/header.html"
    Stop Span "Template/Render/header.html"
  Stop Span "Template/Render/search.html"
Stop Span "view.search.index"
```


`SpanId` needs to be a globally unique string, but otherwise holds no information. A UUID4 is sufficient.


```
{StartSpan: {
              request_id: RequestId,
              span_id: SpanId,
              parent_id: Option<SpanId>,
              operation: String,
              timestamp: Option<EventTimestamp>
            }
}
```

The Timestamp is optional, if left out, it is set to the current time when the Core Agent receives
the command.

### StopSpan

Stops a previously started Span.

The Timestamp is optional, if left out, it is set to the current time when the Core Agent receives
the command.

```
{StopSpan: {
              request_id: RequestId,
              span_id: SpanId,
              timestamp: Option<EventTimestamp>
           }
}
```

### TagSpan

Attaches arbitrary tags to a started Span.

The `Key` may be any `String`, and `Value` may be any JSON serializable structure.

Several keys are treated specially within APM. Other than those keys, the tags are not send onward
to ScoutApm's payload.

```
{TagSpan: {
            request_id: RequestId,
            span_id: SpanId,
            tag: String,
            value: Value,
            timestamp: Option<EventTimestamp>
          }
}
```

#### Known Keys

* `db.statement` - The literal SQL captured from an ORM during an SQL layer. If the original span's
  operation was exactly "SQL/Query", The SQL will be parsed and the operation name of the span will
  change to a more detailed name like "SQL/User/select"

### TagRequest

Attaches arbitrary tags to a started Request.

The Key may be any String, and Value may be any JSON serializable structure.

Any tag attached to a request will appear in ScoutApm's Trace Context page.

```
{TagRequest:  {
                request_id: RequestId,
                tag: String,
                value: String,
                timestamp: Option<EventTimestamp>
              }
}
```

### FinishRequest

Marks a request as finished. Once finished, no new spans, or tags may be attached.

```
{FinishRequest: {
                  request_id: RequestId,
                  timestamp: Option<EventTimestamp>
                }
}
```

### BatchCommand

Allows the agent to send more than a single command at a single time. This can be useful to buffer
entire requests in the Language Agent before sending them as a large block.

**Important** - since the embedded commands are sent at the same time, you will want to mark the actual
timestamp in each of them.

```
{BatchCommand: {
                  commands: Vec<Command>
               }
}
```

### ApplicationEvent

This represents various data about the instrumented application that we want to send up to
the server.

Currently, this is only used in two spots:

* For Language Agent startup information containing application metadata (what language, what
  version of the agent, what libraries are installed, etc)
* For per-minute CPU and Memory details. ("Samplers")

```
{ApplicationEvent:  {
                      event_type: String,
                      event_value: Value,
                      timestamp: EventTimestamp,
                      source: String,
                    }
}
```

#### Application Metadata

At Language Agent startup, it can send a payload with interesting information about the application
that is starting.

```
{'language':           'python',
 'version':            '',
 'server_time':        datetime.utcnow().isoformat() + 'Z',
 'framework':          '',
 'framework_version':  '',
 'environment':        '',
 'app_server':         '',
 'hostname':           '',  // Environment.hostname,
 'database_engine':    '',  // Detected
 'database_adapter':   '',  // Raw
 'application_name':   '',  // Environment.application_name,
 'libraries':          [["Django", '1.11.8'], ...],
 'paas':               '',
 'git_sha':            ''}
```

#### Samplers

By using special strings of `event_type`, the data will be treated as known samplers.

* `event_type` of `"CPU/Utilization"` - the CPU usage of the single Language Agent process. As a
  percent.
* `event_type` of `"Memory/Physical"` - the Memory usage of the single Language Agent process. As a
  number of megabytes

### CoreAgentVersion

Behaves identically to the Unregistered version of this command.

```
{CoreAgentVersion: { }}
```

### CoreAgentShutdown

Behaves identically to the Unregistered version of this command.

```
{CoreAgentShutdown: { }}
```

## Responses

Currently, responses are not used for much. They indicate the success of a command, and the
CoreAgentVersion command replies with the version in its JSON.


## Definitions and Terms

* **Language Agent** - the in-language library collecting metrics (`scout_apm_python`, `scout_apm_ruby`
  for example)
* **Core Agent** - the Rust executable capturing metrics from the language agent
* **Request** - a Request is a single full request, typically initiated by a user. In web frameworks,
  this maps cleanly to a single HTTP request. For background jobs, it is a single definable piece of
  execution work.
* **Span** - A defined range of notable code. A single SQL command, a rendering of a template, or a
  complex bit of compuatation. These are nested, so a "Request Handler" could contain many database
  calls, external requests, and template renderers.
* **RequestId** - a unique String. UUID4 is suitable. For debugging, you may want to prefix it with an
  identifiable string. Existing Language Agents use `req-$uuid`.
* **SpanId** - a unique String. UUID4 is suitable. For debugging, you may want to prefix it with an
  identifiable string. Existing Language Agents use `span-$uuid`.
