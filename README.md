# Scout Core Agent Examples

Using Scout, but have an app in a language Scout doesn't support? You're in luck!

Starting with Scout's Python Monitoring agent, much of the logic has moved to a core agent that acts as a backend to instrumentation. This core agent is implemented as a standalone binary. If you can send messages over a Unix Domain Socket (and you can in every common language), you can add instrumentation to your app!

This repository contains example implementations and documentation for the Core Agent API. 