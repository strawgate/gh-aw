import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock pr_review_buffer before requiring the module
vi.mock("./pr_review_buffer.cjs", () => ({
  setReviewMetadata: vi.fn(),
}));

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("submit_pr_review (Handler Factory Architecture)", () => {
  let handler;
  let setReviewMetadata;

  beforeEach(async () => {
    vi.clearAllMocks();

    setReviewMetadata = (await import("./pr_review_buffer.cjs")).setReviewMetadata;

    const { main } = require("./submit_pr_review.cjs");
    handler = await main({ max: 1 });
  });

  it("should return a function from main()", async () => {
    const { main } = require("./submit_pr_review.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should set review metadata for APPROVE event", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "LGTM! Great changes.",
      event: "APPROVE",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("APPROVE");
    expect(result.body_length).toBe(20);
    expect(setReviewMetadata).toHaveBeenCalledWith("LGTM! Great changes.", "APPROVE");
  });

  it("should set review metadata for REQUEST_CHANGES event", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "Please fix the issues.",
      event: "REQUEST_CHANGES",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("REQUEST_CHANGES");
    expect(setReviewMetadata).toHaveBeenCalledWith("Please fix the issues.", "REQUEST_CHANGES");
  });

  it("should set review metadata for COMMENT event", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "Some general feedback.",
      event: "COMMENT",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("COMMENT");
    expect(setReviewMetadata).toHaveBeenCalledWith("Some general feedback.", "COMMENT");
  });

  it("should normalize event to uppercase", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "Looks good",
      event: "approve",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("APPROVE");
    expect(setReviewMetadata).toHaveBeenCalledWith("Looks good", "APPROVE");
  });

  it("should default event to COMMENT when missing", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "Some feedback",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("COMMENT");
    expect(setReviewMetadata).toHaveBeenCalledWith("Some feedback", "COMMENT");
  });

  it("should reject invalid event values", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "Bad event",
      event: "INVALID",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid review event");
    expect(setReviewMetadata).not.toHaveBeenCalled();
  });

  it("should require body for APPROVE event", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "",
      event: "APPROVE",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Review body is required");
    expect(setReviewMetadata).not.toHaveBeenCalled();
  });

  it("should require body for REQUEST_CHANGES event", async () => {
    const message = {
      type: "submit_pull_request_review",
      body: "",
      event: "REQUEST_CHANGES",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Review body is required");
    expect(setReviewMetadata).not.toHaveBeenCalled();
  });

  it("should allow empty body for COMMENT event", async () => {
    const message = {
      type: "submit_pull_request_review",
      event: "COMMENT",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.event).toBe("COMMENT");
    expect(setReviewMetadata).toHaveBeenCalledWith("", "COMMENT");
  });

  it("should respect max count configuration", async () => {
    const message1 = {
      type: "submit_pull_request_review",
      body: "First review",
      event: "COMMENT",
    };

    const message2 = {
      type: "submit_pull_request_review",
      body: "Second review",
      event: "COMMENT",
    };

    // First call should succeed
    const result1 = await handler(message1, {});
    expect(result1.success).toBe(true);

    // Second call should fail (max=1)
    const result2 = await handler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should not consume max count slot on validation failure", async () => {
    // Invalid event should not consume a slot
    const invalidMessage = {
      type: "submit_pull_request_review",
      body: "Bad event",
      event: "INVALID",
    };

    const result1 = await handler(invalidMessage, {});
    expect(result1.success).toBe(false);

    // Valid message should still succeed since the invalid one didn't count
    const validMessage = {
      type: "submit_pull_request_review",
      body: "Good review",
      event: "APPROVE",
    };

    const result2 = await handler(validMessage, {});
    expect(result2.success).toBe(true);
    expect(result2.event).toBe("APPROVE");
  });

  it("should not consume max count slot when body is missing for APPROVE", async () => {
    // Missing body for APPROVE should not consume a slot
    const noBodyMessage = {
      type: "submit_pull_request_review",
      body: "",
      event: "APPROVE",
    };

    const result1 = await handler(noBodyMessage, {});
    expect(result1.success).toBe(false);

    // Valid message should still succeed
    const validMessage = {
      type: "submit_pull_request_review",
      body: "Now with body",
      event: "APPROVE",
    };

    const result2 = await handler(validMessage, {});
    expect(result2.success).toBe(true);
  });
});
