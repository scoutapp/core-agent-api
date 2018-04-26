/*
 * This example traces a NodeJS request via the Scout Core Agent API. Traces and metrics appear  
 * at Scoutapp.com.
 *
 * Quickstart:
 * 1. Download and extract a core agent binary. URL for OSX: http://s3-us-west-1.amazonaws.com/scout-public-downloads/apm_core_agent/release/scout_apm_core-latest-x86_64-apple-darwin.tgz
 * 2. Start the core agent:
 *    ~/Downloads/scout_apm_core-latest-x86_64-apple-darwin/core-agent start --socket /tmp/core-agent.sock
 * 3. Install dependencies: `npm install uuid bufferpack`
 * 4. Run the app, passing a name for your app and Scout agent key as env vars: `SCOUT_NAME="YOUR APP" SCOUT_KEY="YOUR KEY" node app.js`
 */

const http = require('http');
const net = require('net');
// Used to generate request and span ids
const uuidv1 = require('uuid/v1');
// Used to encode the size of the messages sent via a socket to the core agent
const bufferpack = require('bufferpack');


const hostname = '127.0.0.1';
const port = 4000;

// Communicate w/the Core Agent via this socket
const socket = net.createConnection("/tmp/core-agent.sock");

// Register with the Core Agent
sendToSocket({Register: {app: process.env.SCOUT_NAME,key: process.env.SCOUT_KEY,api_version: '1.0'}});

const server = http.createServer((req, res) => {

  // Start the the trace of the transaction
  var request_id = uuidv1();
  sendToSocket({StartRequest: {request_id: request_id}});

  // Create the span ... traces are composed of spans.
  var span_id = uuidv1();
  // For now, at least of the spans in a transaction must start with 'Controller'
  sendToSocket({StartSpan: {request_id: request_id, span_id: span_id, operation: 'Controller/users/edit' }});

  // The actual work to instrument
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  res.end('Hello World\n');

  // Stop the span and finish the request
  sendToSocket({StopSpan: {request_id: request_id, span_id: span_id}});
  sendToSocket({FinishRequest: {request_id: request_id}});

});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});

// Sends a message (an object) to to the core agent via a Socket
function sendToSocket(object) {
  var message = JSON.stringify(object);
  var size = message.length;
  socket.write(bufferpack.pack('L',[size]));
  socket.write(message);
}

// Encodes a number as a 4 byte big-endian
function toBytesInt32(num) {
  var ascii='';
  for (let i=3;i>=0;i--) {
      ascii+=String.fromCharCode((num>>(8*i))&255);
  }
  return ascii;
};