box.cfg{
	listen=3301
}

local log = require('log')
log.info("app.lua processing start")



counter = require('counter')
counter:start()
