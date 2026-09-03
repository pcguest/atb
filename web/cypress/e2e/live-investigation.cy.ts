describe("ATB investigation against the embedded server", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("loads verified bundle data without API mocks", () => {
    cy.env<{ MOCK_API: boolean | string; SESSION_TOKEN: string }>([
      "MOCK_API",
      "SESSION_TOKEN",
    ]).then(({ MOCK_API, SESSION_TOKEN }) => {
      expect(MOCK_API).to.not.equal(true);
      cy.get('[aria-label="Investigation navigation"]').contains("button", "Trust").click();
      cy.contains("Integrity").should("be.visible");
      cy.contains("Coverage").should("be.visible");
      cy.contains("Corroboration").should("be.visible");
      cy.request({
        url: "/api/v1/bundle/events",
        headers: { "X-ATB-Session-Token": String(SESSION_TOKEN) },
      }).then((response) => {
        expect(response.status).to.equal(200);
        expect(response.body.total).to.be.greaterThan(0);
        expect(response.body.events).to.have.length.greaterThan(0);
      });
    });
  });

  it("passes the strict accessibility audit with live data", () => {
    cy.checkA11yStrict();
  });
});
