# Golang Test Server
___

This is where i'm storing a simple test service that i'm creating as I learn more about socket programming with Golang. 

Right now, Binds a listener to localhost on port 3000, logs to terminal status messages and repeats any message sent to it. 
I may tack on more functionality as I continue learning. 

__

## If you want to run this, it will create a port on localhost:3000
### You can communicate with it by
 - using netcat: ```netcat localhost 3000``` will create a live session. 
 - It should spin off different handlers for individual connections, so you can connect to it more than once. 
 - It doesn't actually do anything, beyond responding with "pong" if you send it "ping"
