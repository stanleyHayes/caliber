import { describe, expect, it } from 'vitest';

import { render } from './entry-server';

describe('entry-server render', () => {
  it('renders the landing page with content and metadata', () => {
    const { html } = render('/');

    expect(html).toContain('<title>Project Caliber</title>');
    expect(html).toContain('Hire on evidence');
    expect(html).toContain('Explainable shortlists');
  });

  it('renders the login page with content and metadata', () => {
    const { html } = render('/login');

    expect(html).toContain('<title>Sign in · Project Caliber</title>');
    expect(html).toContain('Welcome back');
  });

  it('renders the register page with content and metadata', () => {
    const { html } = render('/register');

    expect(html).toContain('<title>Create your account · Project Caliber</title>');
    expect(html).toContain('Passwords must be at least 12 characters');
  });

  it('renders the 404 page with content and metadata', () => {
    const { html } = render('/404');

    expect(html).toContain('<title>Page not found · Project Caliber</title>');
    expect(html).toContain('Not found');
  });
});
