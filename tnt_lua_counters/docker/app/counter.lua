local module = {}


local log = require('log')
log.info("counter.lua processing")


module.start = function(self)
    -- create spaces and indexes
    box.once('init', function()
        box.schema.create_space('counter')
        box.space.counter:create_index(
            "primary", {type = 'tree', parts = {1, 'unsigned'}}
        )
    end)
    return false
end

module.add = function(self, counter_id, num)
	box.space.counter:upsert({counter_id, num}, {{'+', 2, num}})
	return self:get(counter_id)
end

module.get = function(self, counter_id)
	local ts = box.space.counter:select({counter_id})
	return ts[1][2]
end

return module


-- start = function(self)
--     -- create spaces and indexes
--     box.once('init', function()
--         box.schema.create_space('counter')
--         box.space.counter:create_index(
--             "primary", {type = 'tree', parts = {1, 'unsigned'}}
--         )
--     end)
--     return false
-- end

