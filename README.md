# logjam
Accepts submissions of logging information and sends it on to one or more target destinations
like Honeycomb, loggly, cloudwatch, etc.

# HTTP API

## PUT single item as a query

PUT /log?key=value&key2=value2
apikey in x-logjamtoken header

## POST single item as JSON

## POST /log
apikey in x-logjamtoken header
contentType = application/json
body = JSON object with key/value pairs
values can be nested JSON

## POST /log
MAYBE
apikey in x-logjamtoken header
contentType = text/plain
body = key/value pairs separated with newlines where kv are separated by '\w+:\w+'
values can be nested JSON

## POST /multi
apikey in x-logjamtoken header
contentType = application/json
body = array of JSON objects

## GET /config
contentType = application/json
Gets config info:
    array of target configurations:
    number of messages in a couple of buckets


## GET /health
returns 200/ok

# Target Configuration

Each implemented target has a configuration block:

* Name: (a string name for referring to the target)
* Enabled: (bool, false if the target should not be used)
* LastSuccessful: (timestamp of the last successful use of the target)
* (other error info?)
* Triggers: (an array of field names -- if non-empty, only logs with one or more of these fields will be sent to this target)
* Blocks: (an array of field names; the presence of any of them will prevent the target from being used)
* Includes: (array of field names, nonempty means that only the field names specified will possibly be included, unless they are specified in exclude)
* Excludes: (array of field names that should be excluded from the data posted to this target)
