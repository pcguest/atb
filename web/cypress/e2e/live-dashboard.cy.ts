describe("Trust Dashboard against the embedded ATB server", () => {
  beforeEach(() => {
    cy.waitForDashboard();
  });

  it("loads verified bundle data without API mocks", () => {
    cy.env<{ MOCK_API: boolean | string; SESSION_TOKEN: string }>([
      "MOCK_API",
      "SESSION_TOKEN",
    ]).then(({ MOCK_API, SESSION_TOKEN }) => {
      expect(MOCK_API).to.not.equal(true);
      cy.get('[data-testid="viewer-health-value"]').should("not.have.text", "0/100");
      cy.get('[data-testid="chain-length-value"]')
        .invoke("text")
        .then((value) => {
          expect(Number.parseInt(value, 10)).to.be.greaterThan(0);
        });
      cy.request({
        url: "/api/v1/bundle/events",
        headers: {
          "X-ATB-Session-Token": String(SESSION_TOKEN),
        },
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
