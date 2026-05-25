describe("Trust Dashboard", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("displays Trust Score on load", () => {
    cy.get('[data-testid="stats-overview"]').should("be.visible");
    cy.get('[data-testid="trust-score-value"]').should("have.text", "100/100");
    cy.get('[data-testid="event-count-value"]').should("have.text", "3");
    cy.get('[data-testid="chain-length-value"]').should("have.text", "3");
  });

  it("renders trace data and inspector", () => {
    cy.get('[data-testid="event-list"]').should("be.visible");
    cy.contains("Inspector").should("be.visible");
    cy.contains("Mock workflow start").should("be.visible");
  });

  it("keeps event list stable while polling", () => {
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
