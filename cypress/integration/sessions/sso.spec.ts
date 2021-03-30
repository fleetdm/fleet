describe('SSO Sessions', () => {
  // Typically we want to use a beforeEach but not much happens in these tests
  // so sharing some state should be okay and saves a bit of runtime.
  before(() => {
    cy.setup();
    cy.login();
    cy.setupSSO();
    cy.logout();
  });

  // it('Can still login with username/password', () => {
  //   cy.visit('/');
  //   cy.contains(/forgot password/i);

  //   // Log in
  //   cy.get('input').first()
  //     .type('test@fleetdm.com');
  //   cy.get('input').last()
  //     .type('admin123#');
  //   cy.contains('button', 'Login')
  //     .click();

  //   // Verify dashboard
  //   cy.url().should('include', '/hosts/manage');
  //   cy.contains('All Hosts');

  //   // Log out
  //   cy.findByAltText(/user avatar/i)
  //     .click();
  //   cy.contains('button', 'Sign out')
  //     .click();

  //   cy.url().should('match', /\/login$/);
  // });

  it('Can login via SSO', () => {
    cy.visit('/');

    // Log in
    cy.contains('button', 'Sign On With SimpleSAML');

    cy.request({
      method: 'GET',
      url: 'http://localhost:9080/simplesaml/saml2/idp/SSOService.php?spentityid=https://localhost:8080',
      followRedirect: false,
    }).then(response => {
      console.log(response.body);

      const redirect = response.headers['location'];

      cy.request({
        method: 'GET',
        url: redirect,
        // headers,
        followRedirect: false,
      }).then( response => {
        //console.log(response);
        //console.log(response.body);
        // console.log(sessID);

        var el = document.createElement( 'html' );
        el.innerHTML = response.body;
        const authState = el.getElementsByTagName('input')['AuthState'].defaultValue;
        //console.log(authState, redirect);
        
        cy.request({
          method: 'POST',
          url: redirect,
          // headers,
          body: `username=user1&password=user1pass&AuthState=${authState}`,
          followRedirect: false,
        }).then( response => {
          //console.log(response);
          //console.log(response.body);
          const body = `username=user1&password=user1pass&AuthState=${authState}`;  
          console.log('auth', authState);
          console.log('redirect', redirect);
          console.log('body', body);

                  cy.request({
          method: 'POST',
          url: redirect,
          // headers,
          body,
                    form: true,
          followRedirect: false,
        }).then( response => {
          //console.log(response);
          //console.log(response.body);
        });

        });
        
      });
    });
  });
});
