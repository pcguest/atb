describe("ATB investigation against the embedded server", () => {
  let integrityValid = true;

  beforeEach(() => {
    cy.env<{ MOCK_API: boolean | string; SESSION_TOKEN: string }>([
      "MOCK_API",
      "SESSION_TOKEN",
    ]).then(({ MOCK_API, SESSION_TOKEN }) => {
      expect(MOCK_API).to.not.equal(true);
      const headers = { "X-ATB-Session-Token": String(SESSION_TOKEN) };
      cy.request({ url: "/api/v1/verification", headers }).then(({ body }) => {
        integrityValid = body.status === "valid";
        cy.clearLocalStorage("atb-ui-store-v1");
        cy.visit(`/view/#session=${SESSION_TOKEN}`);
        if (!integrityValid) {
          return cy.contains("INTEGRITY CHECK FAILED — HASH CHAIN VERIFICATION FAILED").should("be.visible");
        }
        cy.contains("Local evidence viewer", { timeout: 10000 }).should("be.visible");
        cy.get('nav[aria-label="Investigation"]', { timeout: 10000 })
          .should("be.visible")
          .contains("button", "Evidence status");
        cy.contains("What happened?").should("be.visible");
      });
    });
  });

  it("loads recorded bundle data and enforces the integrity boundary without API mocks", () => {
    cy.env<{ MOCK_API: boolean | string; SESSION_TOKEN: string }>([
      "MOCK_API",
      "SESSION_TOKEN",
    ]).then(({ MOCK_API, SESSION_TOKEN }) => {
      expect(MOCK_API).to.not.equal(true);
      if (!integrityValid) {
        cy.request({
          url: "/api/v1/bundle/events",
          headers: { "X-ATB-Session-Token": String(SESSION_TOKEN) },
          failOnStatusCode: false,
        }).its("status").should("equal", 403);
        cy.contains("Event data is blocked").should("be.visible");
        return;
      }
      cy.get('nav[aria-label="Investigation"]').contains("button", "Evidence status").click();
      cy.contains("Integrity").should("be.visible");
      cy.contains("Coverage").should("be.visible");
      cy.contains("Corroboration").should("be.visible");
      cy.request({
        url: "/api/v1/bundle/events",
        headers: { "X-ATB-Session-Token": String(SESSION_TOKEN) },
      }).then((response) => {
        expect(response.status).to.equal(200);
        expect(response.body.total).to.be.at.least(0);
        expect(response.body.events).to.have.length(response.body.total);
      });
      cy.request({
        url: "/api/v1/investigation/overview",
        headers: { "X-ATB-Session-Token": String(SESSION_TOKEN) },
      }).then(({ body }) => {
        expect(body.integrity_valid).to.be.a("boolean");
        cy.request({
          url: "/api/v1/investigation/findings",
          headers: { "X-ATB-Session-Token": String(SESSION_TOKEN) },
          failOnStatusCode: false,
        }).then((response) => {
          expect(response.status).to.equal(body.integrity_valid ? 200 : 403);
          if (!body.integrity_valid) {
            expect(body.finding_count).to.equal(0);
            expect(body.profile ?? {}).not.to.have.property("coverage_score");
          }
        });
      });
    });
  });

  it("serves library routes without losing the viewer session fragment", () => {
    cy.env<{ SESSION_TOKEN: string }>(["SESSION_TOKEN"]).then(({ SESSION_TOKEN }) => {
      const token = String(SESSION_TOKEN);
      cy.get('nav[aria-label="Library"]').first().contains("Sessions").click();
      cy.location("pathname").should("eq", "/sessions/");
      cy.location("hash").should("eq", `#session=${token}`);
      cy.contains("h1", "Session overview").should("be.visible");
      cy.get('nav[aria-label="Library"]').first().contains("Bundles").click();
      cy.location("pathname").should("eq", "/workspace/");
      cy.location("hash").should("eq", `#session=${token}`);
      cy.contains("h1", "Workspace bundles").should("be.visible");
      cy.contains("ATB Agent").should("be.visible");
    });
  });

  it("passes strict accessibility audits in light theme on library routes", () => {
    cy.env<{ SESSION_TOKEN: string }>(["SESSION_TOKEN"]).then(({ SESSION_TOKEN }) => {
      const token = String(SESSION_TOKEN);
      cy.visit(`/sessions/#session=${token}`);
      cy.contains("h1", "Session overview").should("be.visible");
      cy.get('select[aria-label="Theme"]').select("light");
      cy.checkA11yStrict();
      cy.visit(`/workspace/#session=${token}`);
      cy.contains("h1", "Workspace bundles").should("be.visible");
      cy.get('select[aria-label="Theme"]').select("light");
      cy.checkA11yStrict();
    });
  });

  it("passes the strict accessibility audit with live data", () => {
    if (!integrityValid) {
      cy.checkA11yStrict();
      return;
    }
    for (const surface of ["Run", "Findings", "Timeline", "Context", "Relationships", "Evidence", "Evidence status"]) {
      cy.get('nav[aria-label="Investigation"]').contains("button", surface).click();
      cy.get("main").should("be.visible");
      cy.get("main").should("not.contain.text", "Loading");
      cy.checkA11yStrict();
      cy.screenshot(`investigation-${surface.toLowerCase()}`, { capture: "viewport" });
    }
  });

  it("keeps navigation and workspace within desktop viewports", () => {
    if (!integrityValid) {
      cy.viewport(900, 900);
      cy.document().should((document) => {
        expect(document.documentElement.scrollWidth).to.be.at.most(900);
      });
      return;
    }
    for (const width of [1440, 1280, 1024, 900]) {
      cy.viewport(width, 900);
      for (const surface of ["Run", "Evidence", "Relationships", "Evidence status"]) {
        cy.get('nav[aria-label="Investigation"]').contains("button", surface).click().should("be.visible");
        cy.document().should((document) => {
          expect(document.documentElement.scrollWidth).to.be.at.most(width);
        });
      }
      cy.screenshot(`workspace-${width}`, { capture: "viewport" });
    }
  });

  it("opens and dismisses commands with keyboard focus restoration", () => {
    if (!integrityValid) {
      cy.get('[aria-label="Open command palette"]').should("not.exist");
      return;
    }
    cy.get('button[aria-label="Open command palette"]').focus().click();
    cy.get('[role="dialog"][aria-label="Command palette"]').should("be.visible").within(() => {
      cy.get('[role="combobox"]').should("have.focus").type("{downarrow}");
    });
    cy.checkA11yStrict();
    cy.screenshot("command-palette", { capture: "viewport" });
    cy.focused().type("{esc}");
    cy.get('button[aria-label="Open command palette"]').should("have.focus");
  });
});
