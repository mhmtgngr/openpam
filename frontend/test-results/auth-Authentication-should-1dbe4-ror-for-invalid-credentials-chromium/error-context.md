# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - img [ref=e7]
    - heading "OpenPAM" [level=1] [ref=e9]
    - paragraph [ref=e10]: Sign in to your account
  - generic [ref=e12]:
    - generic [ref=e13]:
      - generic [ref=e14]: Email
      - textbox "Email" [ref=e16]:
        - /placeholder: you@example.com
        - text: invalid@example.com
    - generic [ref=e17]:
      - generic [ref=e18]: Password
      - generic [ref=e19]:
        - textbox "Password" [active] [ref=e20]:
          - /placeholder: ••••••••
          - text: wrongpassword
        - button [ref=e22] [cursor=pointer]:
          - img [ref=e23]
    - button "Sign In" [ref=e26] [cursor=pointer]
  - paragraph [ref=e27]: © 2024 OpenPAM. All rights reserved.
```