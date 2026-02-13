// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { setReviewMetadata } = require("./pr_review_buffer.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "submit_pull_request_review";

/** @type {Set<string>} Valid review event types */
const VALID_EVENTS = new Set(["APPROVE", "REQUEST_CHANGES", "COMMENT"]);

/**
 * Main handler factory for submit_pull_request_review
 * Returns a message handler that stores review metadata (body and event)
 * in the shared pr_review_buffer. The actual review submission happens
 * during the handler manager's finalization step.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 1;

  core.info(`Submit PR review handler initialized: max=${maxCount}`);

  let processedCount = 0;

  /**
   * Message handler that stores review metadata
   * @param {Object} message - The submit_pull_request_review message
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs
   * @returns {Promise<Object>} Result with success status
   */
  return async function handleSubmitPRReview(message, resolvedTemporaryIds) {
    if (processedCount >= maxCount) {
      core.warning(`Skipping submit_pull_request_review: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    // Validate event field
    const event = (message.event || "COMMENT").toUpperCase();
    if (!VALID_EVENTS.has(event)) {
      core.warning(`Invalid review event: ${message.event}. Must be one of: APPROVE, REQUEST_CHANGES, COMMENT`);
      return {
        success: false,
        error: `Invalid review event: ${message.event}. Must be one of: APPROVE, REQUEST_CHANGES, COMMENT`,
      };
    }

    // Body is required for APPROVE and REQUEST_CHANGES
    const body = message.body || "";
    if ((event === "APPROVE" || event === "REQUEST_CHANGES") && !body) {
      core.warning(`Review body is required for event type ${event}`);
      return {
        success: false,
        error: `Review body is required for event type ${event}`,
      };
    }

    // Only increment after validation passes
    processedCount++;

    core.info(`Setting review metadata: event=${event}, bodyLength=${body.length}`);

    // Store the review metadata in the shared buffer
    setReviewMetadata(body, event);

    return {
      success: true,
      event: event,
      body_length: body.length,
    };
  };
}

module.exports = { main };
