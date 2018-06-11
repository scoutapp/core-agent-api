<?php
    /*
     * This example traces a PHP request via the Scout Core Agent API. Traces and metrics appear  
     * at Scoutapp.com.
     *
     * Quickstart:
     * 1. Download and extract a core agent binary. URL for OSX: http://s3-us-west-1.amazonaws.com/scout-public-downloads/apm_core_agent/release/scout_apm_core-latest-x86_64-apple-darwin.tgz
     * 2. Start the core agent: `~/Downloads/scout_apm_core-latest-x86_64-apple-darwin/core-agent start --socket /tmp/core-agent.sock`
     * 3. Run the app, passing a name for your app and Scout agent key as env vars: `env SCOUT_NAME="YOUR APP" SCOUT_KEY="YOUR KEY" php -d variables_order=EGPCS -S 127.0.0.1:4000 HelloWorld.php`
     * 4. Hit the listening http endpoint: `curl localhost:4000`
     */

    $sock = socket_create(AF_UNIX, SOCK_STREAM, 0);
    socket_connect($sock, '/tmp/core-agent.sock');
    socket_set_nonblock($sock);

    sendToSocket($sock, ['Register' => ['app' => $_ENV['SCOUT_NAME'], 'key' => $_ENV['SCOUT_KEY'], 'api_version' => '1.0']]);

    $request_id = 'req-'.uuid4();
    sendToSocket($sock, ['StartRequest' => ['request_id' => $request_id]]);

    $span_id = 'span-'.uuid4();
    sendToSocket($sock, ['StartSpan' => ['request_id' => $request_id, 'span_id' => $span_id, 'operation' => 'Controller/HelloWorld']]);

    echo "Hello World!";

    sendToSocket($sock, ['StopSpan' => ['request_id' => $request_id, 'span_id' => $span_id]]);
    sendToSocket($sock, ['FinishRequest' => ['request_id' => $request_id]]);

    function sendToSocket($sock, $arr) {
        $message = json_encode($arr);
        $size = strlen($message);
        socket_send($sock, pack('N', $size), 4, 0);
        socket_send($sock, $message, $size, 0);
    }

    # Original uuid4 source: http://www.php.net/manual/en/function.uniqid.php#94959
    function uuid4() 
    {
        return sprintf('%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
        mt_rand(0, 0xffff), mt_rand(0, 0xffff),
        mt_rand(0, 0xffff),
        mt_rand(0, 0x0fff) | 0x4000,
        mt_rand(0, 0x3fff) | 0x8000,
        mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
        );
    }
?>
