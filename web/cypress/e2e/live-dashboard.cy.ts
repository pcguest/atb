describe("Trust Dashboard against the embedded ATB server", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("loads verified bundle data without API mocks", () => {
    expect(Cypress.env("MOCK_API")).to.not.equal(true);
    cy.get('[data-testid="viewer-health-value"]').should("not.have.text", "0/100");
    cy.get('[data-testid="chain-length-value"]').invoke("text").then((value) => {
      expect(Number.parseInt(value, 10)).to.be.greaterThan(0);
    });
    cy.request({
      url: "/api/v1/bundle/events",
      headers: {
        "X-ATB-Session-Token": String(Cypress.env("SESSION_TOKEN")),
      },
    }).then((response) => {
      expect(response.status).to.equal(200);
      expect(response.body.total).to.be.greaterThan(0);
      expect(response.body.events).to.have.length.greaterThan(0);
    });
  });

  it("passes the strict accessibility audit with live data", () => {
    cy.checkA11yStrict();
  });
});
