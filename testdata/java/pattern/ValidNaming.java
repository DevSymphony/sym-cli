/**
 * Test file with correct naming conventions
 * Complies with Checkstyle naming rules
 */
package com.example;

public class ValidNaming {

    // Correct: Constant in UPPER_SNAKE_CASE
    private static final String API_KEY = "from-environment";

    // Correct: Variable in camelCase
    private int goodVariable = 100;

    // Correct: Method in camelCase
    public void goodMethod() {
        System.out.println("This method has good naming");
    }

    // Correct: Parameter in camelCase
    public String processData(String userName) {
        return "Hello " + userName;
    }

    // Correct: Main method
    public static void main(String[] args) {
        ValidNaming obj = new ValidNaming();
        obj.goodMethod();
    }
}
