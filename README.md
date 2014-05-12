pingPongStatus
==============

Using a spark.io device and a motion sensor to let people know when the ping pong table is free  

Details
--
A passive infrared sensor (PIR) from Radioshack was used http://www.radioshack.com/product/index.jsp?productId=28386046  
The PIR was connected to the 3.3V, ground, and D0 pins on the Spark core

The Spark keeps polling the PIR sensor and setting the lastMotion variable to 0 when it detects motion, otherwise it increments it every second.
Using the http://docs.spark.io/#/firmware/data-and-control-spark-variable mechanism, I'm able to query the current value from the Spark cloud service.

This app provides a simple web server that hits the Spark API and lets people know if the ping pong table is available or busy.







