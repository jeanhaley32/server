# Golang Test Server
___

This is where i'm storing a simple test service that i'm creating as I learn more about socket programming with Golang. 

Right now, This script binds a listener to localhost on an arbitrary port, and then logs to terminal some status messages, and echo back any messages sent to it. 
I may tack on more functionality as I continue learning. 

__

## If you want to run this, it will start a listener on localhost and log the socket to terminal so you can connect to it. 
### You can communicate with it by
 - using netcat: ```netcat localhost 3000``` (if port is 3000) will create a live session. 
 - It should spin off different handlers for individual connections, so you can connect to it more than once. 
 - It doesn't actually do anything, beyond responding with "pong" if you send it "ping"
