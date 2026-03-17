describe("ATB Trust Dashboard", () => {
  it("load bundle -> display Trust Score", () => {
    cy.visit("/view/");
    cy.contains("Trust Score").should("be.visible");
    cy.contains("Verify bundle").should("be.visible");
  });
});
