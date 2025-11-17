/**
 * Test file with valid length constraints
 * All lines, methods, and parameter counts are within limits
 */
package com.example;

public class ValidLength {

    private static final String CONFIG = "config-value";

    // Correct: 4 parameters or fewer
    public String formatUser(String name, String email, String role) {
        return name + " (" + email + ") - " + role;
    }

    // Correct: Short, focused method
    public int add(int a, int b) {
        return a + b;
    }

    // Correct: Method within reasonable length
    public void processRequest() {
        String input = readInput();
        String validated = validate(input);
        String result = transform(validated);
        save(result);
    }

    private String readInput() {
        return "input";
    }

    private String validate(String input) {
        if (input == null || input.isEmpty()) {
            throw new IllegalArgumentException("Invalid input");
        }
        return input;
    }

    private String transform(String data) {
        return data.toUpperCase();
    }

    private void save(String data) {
        System.out.println("Saved: " + data);
    }

    public static void main(String[] args) {
        ValidLength obj = new ValidLength();
        obj.processRequest();
    }
}
