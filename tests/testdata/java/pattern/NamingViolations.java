/**
 * Test file with naming convention violations
 * Violates Checkstyle naming rules
 */
package com.example;

// VIOLATION: Class name should start with uppercase (PascalCase)
public class invalidClassName {

    // VIOLATION: Constant should be UPPER_SNAKE_CASE
    private static final String apiKey = "sk-1234567890";

    // VIOLATION: Variable name using UPPER_SNAKE_CASE (should be camelCase)
    private int BAD_VARIABLE = 100;

    // VIOLATION: Method name starts with uppercase (should be camelCase)
    public void BadMethod() {
        System.out.println("This method has bad naming");
    }

    // VIOLATION: Method parameter uses snake_case
    public String processData(String user_name) {
        return "Hello " + user_name;
    }

    // VIOLATION: Multiple naming issues
    public static void MAIN(String[] args) {
        invalidClassName obj = new invalidClassName();
        obj.BadMethod();
    }
}
