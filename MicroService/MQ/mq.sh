while [ 1 ];
do
    curl --location --request POST '127.0.0.1:5150/put?topic=test' --header 'Content-Type: text/plain' --data-raw 'hello'
done