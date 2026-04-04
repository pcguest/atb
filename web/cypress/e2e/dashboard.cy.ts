describe("Trust Dashboard", () => {
  const selectRole = (role: "engineer" | "auditor" | "executive") => {
    cy.get('[data-testid="role-selector"]').click({ force: true });
    cy.get(`[data-testid="role-selector-${role}"]`).click({ force: true });
  };

  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("displays Trust Score on load", () => {
    cy.get('[data-testid="trust-score-card"]').should("be.visible");
    cy.get('[data-testid="trust-score-value"]', { timeout: 15000 })
      .should("be.visible")
      .should("not.have.text", "Loading...")
      .should("not.have.text", "N/A")
      .invoke("text")
      .then((value) => {
        expect(value.trim()).to.match(/^\d+$|TEST-MODE$/);
      });
  });

  it("switches roles and updates UI", () => {
    selectRole("engineer");
    cy.get('[data-testid="raw-data-inspector"]').should("be.visible");

    selectRole("auditor");
    cy.get('[data-testid="raw-data-inspector"]').should("not.exist");
    cy.get('[data-testid="export-evidence-btn"]').should("be.visible");

    selectRole("executive");
    cy.get('[data-testid="executive-summary"]').should("be.visible");
  });

  it("keeps event list stable while polling", () => {
    selectRole("engineer");
    cy.get('[data-testid="event-list"]').should("be.visible");

    cy.get("body").then(($body) => {
      const initialCount = $body.find('[data-testid="event-list-item"]').length;

      cy.wait(6000);

      cy.get("body").then(($nextBody) => {
        const afterPollingCount = $nextBody.find('[data-testid="event-list-item"]').length;
        expect(afterPollingCount).to.be.gte(initialCount);
      });
    });
  });

  it("passes accessibility audit", () => {
    cy.checkA11yStrict();
  });
});
