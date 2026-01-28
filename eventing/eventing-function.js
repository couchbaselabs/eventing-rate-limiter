// Enforce rate limits on every incoming request on a per-user basis using a rate limiter
function OnUpdate(doc, meta, xattrs) {
    log("OnUpdate function has started running!");

    const user_id = doc.user_id;

    let done = false;
    while (!done) {
        // Get the tier of the `user_id`
        let userAccountsMeta = {
            "id": user_id
        };
        let userAccountsResult = couchbase.get(userAccounts, userAccountsMeta, {
            "cache": true
        });
        if (!userAccountsResult.success) {
            throw new Error("Error(Unable to get the user's details): " + JSON.stringify(userAccountsResult));
        }
        const tier = userAccountsResult.doc.tier;

        // Get the rate limit for the tier
        let tierLimitsMeta = {
            "id": "limits"
        };
        let tierLimitsResult = couchbase.get(tierLimits, tierLimitsMeta, {
            "cache": true
        });
        if (!tierLimitsResult.success) {
            throw new Error("Error(Unable to get the tier limits): " + JSON.stringify(tierLimitsResult));
        }
        const rateLimit = tierLimitsResult.doc[tier];

        // Try to get the rate limit count for the `user_id`
        const userIDMeta = {
            "id": user_id
        };
        const result = couchbase.get(rateLimiter, userIDMeta);

        // If the rate limit count for the `user_id` does not exist. Try to create it.
        while (!result.success) {
            couchbase.insert(rateLimiter, userIDMeta, {
                "count": 0
            });
            result = couchbase.get(rateLimiter, userIDMeta);
        }

        // Assign the counter document's `count` and `meta` to local variables for convenience
        const counterDocCount = result.doc.count;
        const counterDocMeta = result.meta;

        // Check if the counter has hit the rate limit
        // We use >= instead of == to handle the edge case where the tier limits have reduced
        // but the tier tracker documents have not yet been deleted.
        if (counterDocCount >= rateLimit) {
            log("User with ID '" + user_id + "' hit their rate limit of " + rateLimit + "!");
            done = true;
            continue;
        }

        // Update the count in the document
        let res = couchbase.mutateIn(rateLimiter, counterDocMeta, [
            couchbase.MutateInSpec.replace("count", counterDocCount + 1),
        ]);
        done = res.success;
        if (done) {
            // POST the request to the `llmEndpoint`
            delete doc.user_id;
            const response = curl('POST', llmEndpoint, doc);
            if (response.status != 200) {
                throw new Error("Error(MyLLM endpoint is not working): " + response.status);
            }
        }
    }

    log("OnUpdate function has finished running!");
}

// We don't need the OnDelete function for this use case
// function OnDelete(meta, options) {
// }

// Setup the rate limiter
function OnDeploy(action) {
    log("OnDeploy function has started running!" + JSON.stringify(action));

    // GET the tiers from the `tiersEndpoint`
    const response = curl('GET', tiersEndpoint);
    if (response.status != 200) {
        throw new Error("Error(Cannot get tiers): " + JSON.stringify(response));
    }
    const tiers = response.body;
    log("Successfully retrieved the tiers: " + JSON.stringify(tiers));

    // Write the tiers to the `tierLimits` keyspace, in the document with ID `limits`
    tierLimits["limits"] = tiers;

    // If we are deploying, then we should delete all the existing document in the keyspace `rateLimiter`
    if (action.reason === "deploy") {
        let results = N1QL("DELETE FROM `rate-limiter`.`my-llm`.tracker");
        results.close();
        log("Deleted all the documents in the `rate-limiter`.`my-llm`.tracker keyspace as we are deploying!");
    }

    // Create a timer to run every 24 hours to refresh the tiers
    let timeAfter24hours = new Date();
    timeAfter24hours.setDate(timeAfter24hours.getDate() + 1);
    log("Time after 24 hours is: " + timeAfter24hours);

    createTimer(updateTierCallback, timeAfter24hours, "tier-updater", {});

    // Create a timer to run every 1 hour to reset user rate limits
    let timeAfter1Hour = new Date();
    timeAfter1Hour.setHours(timeAfter1Hour.getHours() + 1);
    log("Time after 1 hour is: " + timeAfter1Hour);

    createTimer(resetRateLimiter, timeAfter1Hour, "rate-limit-resetter", {});

    log("OnDeploy function has finished running!" + JSON.stringify(action));
}

// Helper functions

// Function to reset the rate limits for all users every 1 hour
function resetRateLimiter(context) {
    log('From resetRateLimiter: timer fired');

    let results = N1QL("DELETE FROM `rate-limiter`.`my-llm`.tracker");
    results.close();

    // Create a timer to run every 1 hour to reset user rate limits
    let timeAfter1Hour = new Date();
    timeAfter1Hour.setHours(timeAfter1Hour.getHours() + 1);
    log("Time after 1 hour is: " + timeAfter1Hour);

    createTimer(resetRateLimiter, timeAfter1Hour, "rate-limit-resetter", {});
}

// Function to update the user tiers every 24 hours
function updateTierCallback(context) {
    log('From updateTierCallback: timer fired');

    // GET the tiers from the `tiersEndpoint`
    const response = curl('GET', tiersEndpoint);
    if (response.status != 200) {
        log("Error(Cannot get tiers): " + JSON.stringify(response));
    } else {
        const tiers = response.body;
        log("Successfully retrieved the tiers: " + JSON.stringify(tiers));

        // Write the tiers to the `tierLimits` keyspace, in the document with ID `limits`
        tierLimits["limits"] = tiers;
    }

    // Create a timer to run every 24 hours to refresh the tiers
    let timeAfter24hours = new Date();
    timeAfter24hours.setDate(timeAfter24hours.getDate() + 1);
    log("Time after 24 hours is: " + timeAfter24hours);

    createTimer(updateTierCallback, timeAfter24hours, "tier-updater", {});
}
