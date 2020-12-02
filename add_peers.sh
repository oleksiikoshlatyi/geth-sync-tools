#!/bin/sh

#read first argument, should be file with enode addresses
if [ -z "$1" ]; then 
    filename='nodes.txt' #default value
else
    filename=$1
fi

#read second argument, should be URL to node http endpoint
if [ -z "$2" ]; then
    node_URL='http://sandbox:8545' #default value
else
    node_URL=$2
fi

echo "Reading from $filename nodes and add this to $node_URL..."

#Adding node address from list as a peer to node:
counter=0 #counter for succeed added peers
while read PEER
do  
    echo $PEER
    if 
        docker run --rm ethereum/client-go --exec "admin.addPeer('$PEER')" attach $node_URL ; 
    then
        counter=$((counter+1)) ; echo "node #$counter added"
    else
        echo "Failed to add node"
    fi
done < $filename
