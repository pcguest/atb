import "cypress-axe";

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

    cy.intercept("GET", "**/api/v1/investigation/overview*", {
      statusCode: 200,
      body: {
        bundle_path: "/tmp/mock-bundle.atb",
        event_count: 3,
        integrity_valid: true,
        integrity_status: "VERIFIED",
        profile: null,
        finding_count: 1,
        critical_findings: 0,
        custody_state: "Local only",
        summary: "One bounded finding requires review.",
      },
    }).as("getInvestigationOverview");

    cy.intercept("GET", "**/api/v1/investigation/trust*", {
      statusCode: 200,
      body: {
        proof_statement: "ATB proves the integrity and order of records presented in a bundle.",
        integrity_valid: true,
        canonicalisation: "rfc8785",
        signature_status: "absent",
        anchor_status: "absent",
        profile_pass: false,
        coverage_score: 0,
        coverage_grade: "",
        assessment_coverage: 0,
        assurance_valid: false,
        external_corroboration: false,
        custody_state: "Local only",
        limitations: ["ATB does not prove complete capture."],
      },
    }).as("getInvestigationTrust");

    cy.intercept("GET", "**/api/v1/investigation/findings*", {
      statusCode: 200,
      body: {
        findings: [
          {
            flag: "tool_without_approval",
            severity: "high",
            title: "No matching approval",
            detail: "No matching earlier approval exists in the captured evidence.",
            basis: "captured session evidence",
            boundedness: "captured_evidence_only",
            what_atb_can_conclude: "No captured match exists.",
            what_atb_cannot_conclude: "ATB cannot prove universal absence.",
            event_seqs: [2],
          },
        ],
      },
    }).as("getInvestigationFindings");

    cy.intercept("GET", "**/api/v1/investigation/timeline*", {
      statusCode: 200,
      body: { events: [] },
    }).as("getInvestigationTimeline");

    cy.intercept("GET", "**/api/v1/investigation/context*", {
      statusCode: 200,
      body: { lineage: { units: [], operations: [], warnings: [] }, capabilities: [] },
    }).as("getInvestigationContext");

    cy.intercept("GET", "**/api/v1/investigation/relationships*", {
      statusCode: 200,
      body: { relationships: [] },
    }).as("getInvestigationRelationships");
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
      cy.wait(["@getVerification", "@getInvestigationOverview"]);
    }
    cy.contains("ATB View", { timeout: 10000 }).should("be.visible");
    cy.get('[aria-label="Investigation navigation"]', { timeout: 10000 })
      .should("be.visible")
      .contains("button", "Trust");
    cy.contains("What happened?").should("be.visible");
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
      const messages = violations.map((violation) => {
        const msg = `${violation.id}: ${violation.help} :: ${violation.nodes
          .map((node) => node.target.join(", "))
          .join(" | ")}`;
        Cypress.log({ name: "axe", message: msg });
        console.error("[axe]", msg);
        return msg;
      });
      if (messages.length > 0) throw new Error(messages.join("\n"));
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
