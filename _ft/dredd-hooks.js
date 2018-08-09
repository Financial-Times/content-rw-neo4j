var hooks = require('hooks');

hooks.beforeEach(function (transaction) {
    if (transaction.name.startsWith("Health > /__gtg")) {
        hooks.log("skipping: " + transaction.name);
        transaction.skip = true;
    }
});