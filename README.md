pingPongStatus
==============

Using a [Spark Core](http://spark.io) device and a motion sensor to let people know when the ping pong table is available.

Details
--
A passive infrared sensor (PIR) from Radioshack was used http://www.radioshack.com/product/index.jsp?productId=28386046  
The PIR was connected to the 3.3V, ground, and D0 pins on the Spark core

The Spark device keeps polling the PIR sensor and sends events up to the spark.io cloud when the table is busy, or when it was busy and detects it has been free for 60 seconds.  
See: http://docs.spark.io/api/#reading-data-from-a-core-events  

This app provides a simple web server that listens for busy/free events and lets people know the status.







