import "cypress-axe";

Cypress.on("uncaught:exception", (err) => {
  console.warn("Uncaught exception:", err.message);
  return false;
});

Cypress.Commands.add("retryableRequest", (options: Partial<Cypress.RequestOptions> = {}) => {
  return cy.request({
    ...options,
    failOnStatusCode: false,
    retryOnNetworkFailure: true,
    retryOnStatusCodeFailure: true,
  });
});

beforeEach(() => {
  return cy.env<{ MOCK_API: boolean | string }>(["MOCK_API"]).then(({ MOCK_API }) => {
    const isMockMode = MOCK_API === true || MOCK_API === "true";
    if (!isMockMode) {
      return;
    }

    const now = new Date().toISOString();
    const genesisHash = "0".repeat(64);
    const headHash = "a".repeat(64);

    cy.intercept("GET", "**/api/v1/verification*", {
      statusCode: 200,
      body: {
        status: "valid",
        message: "Mock verification passed",
        bundle_path: "/tmp/mock-bundle.atb",
        chain_length: 3,
        head_hash: headHash,
      },
    }).as("getVerification");

    cy.intercept("GET", "**/api/v1/bundle/meta*", {
      statusCode: 200,
      body: {
        bundle_path: "/tmp/mock-bundle.atb",
        event_count: 3,
        type_counts: {
          "ai.chain.run": 1,
          "ai.request.received": 1,
          "ai.tool.exec": 1,
        },
        first_timestamp: now,
        last_timestamp: now,
        genesis_hash: genesisHash,
        verified_at: now,
        verified: true,
        verification_message: "Mock metadata loaded",
      },
    }).as("getBundleMeta");

    cy.intercept("GET", "**/api/v1/bundle/events*", {
      statusCode: 200,
      body: {
        offset: 0,
        limit: 200,
        total: 3,
        events: [
          {
            seq: 1,
            type: "ai.request.received",
            hash: genesisHash,
            prev_hash: "f".repeat(64),
            timestamp: now,
            data: { name: "Mock workflow start" },
          },
          {
            seq: 2,
            type: "ai.tool.exec",
            hash: "b".repeat(64),
            prev_hash: genesisHash,
            timestamp: now,
            data: { tool: "search", status: "ok" },
          },
          {
            seq: 3,
            type: "ai.chain.run",
            hash: headHash,
            prev_hash: "b".repeat(64),
            timestamp: now,
            data: { result: "complete" },
          },
        ],
      },
    }).as("getBundleEvents");

    cy.intercept("GET", "**/api/v1/bundle/graph*", {
      statusCode: 200,
      body: {
        nodes: [
          { id: "evt-1", label: "Start", type: "chain", event_type: "ai.chain.start" },
          { id: "evt-2", label: "Tool", type: "tool", event_type: "ai.tool.call" },
          { id: "evt-3", label: "End", type: "chain", event_type: "ai.chain.end" },
        ],
        edges: [
          { id: "edge-1", source: "evt-1", target: "evt-2", label: "next" },
          { id: "edge-2", source: "evt-2", target: "evt-3", label: "next" },
        ],
      },
    }).as("getBundleGraph");
  });
});

Cypress.Commands.add("waitForDashboard", () => {
  cy.env<{ MOCK_API: boolean | string; SESSION_TOKEN?: string }>([
    "MOCK_API",
    "SESSION_TOKEN",
  ]).then(({ MOCK_API, SESSION_TOKEN }) => {
    const isMockMode = MOCK_API === true || MOCK_API === "true";
    const sessionToken = String(SESSION_TOKEN ?? "");
    cy.clearLocalStorage("atb-ui-store-v1");
    cy.visit(!isMockMode && sessionToken ? `/view/#session=${sessionToken}` : "/view");
    if (isMockMode) {
      cy.wait(["@getVerification", "@getBundleMeta", "@getBundleEvents", "@getBundleGraph"]);
    }
    cy.get('[data-testid="stats-overview"]', { timeout: 10000 }).should("be.visible");
    cy.get('[data-testid="event-list"]', { timeout: 10000 }).should("be.visible");
    cy.get('[data-testid="viewer-health-value"]', { timeout: 10000 })
      .should("be.visible")
      .should("not.contain.text", "TEST-MODE");
  });
});

type A11yContext = Parameters<typeof cy.checkA11y>[0];

Cypress.Commands.add("checkA11yStrict", (context?: A11yContext) => {
  cy.injectAxe();
  cy.checkA11y(
    context,
    {
      includedImpacts: ["critical", "serious", "moderate", "minor"],
      rules: {
        "color-contrast": { enabled: true },
        "button-name": { enabled: true },
        "link-name": { enabled: true },
        "aria-required-attr": { enabled: true },
      },
    },
    (violations) => {
      violations.forEach((violation) => {
        const msg = `${violation.id}: ${violation.help} :: ${violation.nodes
          .map((node) => node.target.join(", "))
          .join(" | ")}`;
        Cypress.log({ name: "axe", message: msg });
        console.error("[axe]", msg);
      });
    },
  );
});

declare global {
  namespace Cypress {
    interface Chainable {
      retryableRequest(options?: Partial<RequestOptions>): Chainable<Response<unknown>>;
      waitForDashboard(): Chainable<void>;
      checkA11yStrict(context?: A11yContext): Chainable<void>;
    }
  }
}

export {};
