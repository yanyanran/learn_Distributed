i=1
 while [ $i -le 5 ]
 do
   curl --location --request POST '127.0.0.1:5150/put?topic=test' --header 'Content-Type: text/plain' --data-raw $i
   i=$[$i+1]
 done