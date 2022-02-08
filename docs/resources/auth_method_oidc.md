---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "akeyless_auth_method_oidc Resource - terraform-provider-akeyless"
subcategory: ""
description: |-
  OIDC Auth Method Resource
---

# akeyless_auth_method_oidc (Resource)

OIDC Auth Method Resource



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **name** (String) Auth Method name
- **unique_identifier** (String) A unique identifier (ID) value should be configured for OIDC, OAuth2, LDAP and SAML authentication method types and is usually a value such as the email, username, or upn for example. Whenever a user logs in with a token, these authentication types issue a sub claim that contains details uniquely identifying that user. This sub claim includes a key containing the ID value that you configured, and is used to distinguish between different users from within the same organization.

### Optional

- **access_expires** (Number) Access expiration date in Unix timestamp (select 0 for access without expiry date)
- **access_id** (String) Auth Method access ID
- **allowed_redirect_uri** (Set of String) Allowed redirect URIs after the authentication (default is https://console.akeyless.io/login-oidc to enable OIDC via Akeyless Console and  http://127.0.0.1:* to enable OIDC via akeyless CLI)
- **bound_ips** (Set of String) A CIDR whitelist with the IPs that the access is restricted to
- **client_id** (String) Client ID
- **client_secret** (String) Client Secret
- **force_sub_claims** (Boolean) enforce role-association must include sub claims
- **id** (String) The ID of this resource.
- **issuer** (String) Issuer URL

