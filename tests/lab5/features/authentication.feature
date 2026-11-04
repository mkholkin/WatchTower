@lab5 @authentication
Feature: Real HTTP authentication requires a second factor
  Each scenario uses fresh disposable accounts and secrets generated only in memory.
  Mailpit receives real SMTP messages; PostgreSQL timestamps control expiry.

  Background:
    Given a fresh registered account

  @REQ-MFA
  Scenario: Password alone grants no session and the delivered code completes login
    When the account submits its correct password
    Then only a challenge is returned
    And monitors reject absent and invalid credentials
    When the account submits its delivered login code
    Then a full session can read monitors

  @REQ-CREDENTIALS
  Scenario: Unknown credentials and an incorrect password are rejected
    When unknown credentials are submitted
    Then the response status is 401
    When the account submits an incorrect password
    Then the response status is 401

  @REQ-LOCK @REQ-RECOVERY
  Scenario: Three password failures lock the account until recovery
    Given the account has a full session
    When three incorrect passwords are submitted
    Then the account is locked and its prior session is rejected
    When the account recovers with its delivered token and a fresh password
    Then the old password is rejected and the fresh password completes MFA

  @REQ-LOCK
  Scenario: A correct password does not reset the combined failure counter
    When two incorrect passwords are submitted
    And the account submits its correct password
    And the account submits an incorrect login code
    Then the response status is 423
    And the account is locked

  @REQ-LOCK
  Scenario: Three incorrect second factors lock the account
    When the account submits its correct password
    And three incorrect login codes are submitted
    Then the account is locked

  @REQ-CONCURRENCY @REQ-LOCK
  Scenario: Concurrent password failures cannot bypass the limit
    When six incorrect passwords are submitted concurrently
    Then the account is locked with exactly three recorded failures

  @REQ-OTP
  Scenario: A consumed login code cannot be replayed
    Given the account has a full session
    When the account submits its delivered login code again
    Then the response status is 401

  @REQ-OTP
  Scenario: An expired login code is rejected
    When the account submits its correct password
    And its login code expires in the database
    And the account submits its delivered login code
    Then the response status is 410

  @REQ-CONCURRENCY @REQ-OTP
  Scenario: Concurrent code redemption creates only one session
    When the account submits its correct password
    And the delivered login code is submitted twice concurrently
    Then exactly one verification succeeds

  @REQ-OTP
  Scenario: A new login challenge invalidates the previous challenge
    When the account submits its correct password
    And the account replaces its login challenge
    And the account submits its previous challenge and code
    Then the response status is 401

  @REQ-RECOVERY
  Scenario: A recovery token can be consumed only once
    When the account recovers with its delivered token and a fresh password
    And the consumed recovery token is submitted again
    Then the response status is 401

  @REQ-RECOVERY
  Scenario: An expired recovery token is rejected
    When the account requests recovery
    And its recovery token expires in the database
    And the account submits its recovery token with a fresh password
    Then the response status is 410

  @REQ-RECOVERY
  Scenario: Incorrect recovery tokens cannot change the password
    When the account requests recovery
    And an incorrect recovery token is submitted
    Then the response status is 401
    When the account completes MFA with its current password
    Then a full session can read monitors

  @REQ-RECOVERY
  Scenario: Unknown recovery requests have the same public response
    When known and unknown accounts request recovery
    Then both recovery responses are accepted and disclose no secrets

  @REQ-OWNERSHIP
  Scenario: Another account cannot redeem the recovery token
    Given another fresh registered account
    When the account requests recovery
    And its recovery token is submitted for the other account
    Then the response status is 401
    When the other account completes MFA with its original password
    Then the other account has a full session

  @REQ-ROTATION
  Scenario: An expired password requires changing before accessing monitors
    Given the account has a full session
    And the password becomes older than ninety days
    Then the prior full session is rejected
    When the account completes MFA with its current password
    Then the session permits password change but cannot read monitors
    When the account changes its password using its session
    Then all prior sessions are rejected
    And the old password is rejected and the fresh password completes MFA

  @REQ-CHANGE
  Scenario: Voluntary password change revokes sessions and outstanding challenges
    Given the account has a full session
    When the account opens another login challenge
    And the account changes its password using its session
    Then all prior sessions are rejected
    When the account submits its delivered login code
    Then the response status is 401
    And the old password is rejected and the fresh password completes MFA

  @REQ-RECOVERY @REQ-REVOKE
  Scenario: Recovery revokes existing sessions and outstanding challenges
    Given the account has a full session
    When the account opens another login challenge
    And the account recovers with its delivered token and a fresh password
    Then all prior sessions are rejected
    When the account submits its delivered login code
    Then the response status is 401

  @REQ-POLICY
  Scenario Outline: Registration rejects passwords outside byte length limits
    When a fresh account registers with a password of <length> bytes
    Then the response status is 400
    Examples:
      | length |
      | 0      |
      | 73     |

  @REQ-POLICY
  Scenario Outline: Registration accepts passwords at byte length boundaries
    When a fresh account registers with a password of <length> bytes
    Then the response status is 201
    Examples:
      | length |
      | 1      |
      | 72     |

  @REQ-POLICY
  Scenario: Password change rejects unchanged and empty passwords
    Given the account has a full session
    When the account attempts to reuse its current password
    Then the response status is 400
    When the account attempts to change to a empty password
    Then the response status is 400
    And a full session can read monitors

  @REQ-POLICY
  Scenario: Recovery cannot set an unchanged or empty password
    When the account requests recovery
    And recovery is attempted with the unchanged password
    Then the response status is 400
    When recovery is attempted with a empty password
    Then the response status is 400
    When the account submits its recovery token with a fresh password
    Then the response status is 204

  @REQ-CHANGE
  Scenario: Password change requires the correct current password
    Given the account has a full session
    When password change is attempted with an incorrect current password
    Then the response status is 401
    When the account completes MFA with its current password
    Then a full session can read monitors

  @REQ-LEGACY
  Scenario: Legacy password routes cannot bypass MFA
    When legacy registration and login are attempted
    Then neither legacy route creates an authenticated session

  @REQ-REGISTRATION
  Scenario: Duplicate registration is rejected
    When the account registers again
    Then the response status is 409
