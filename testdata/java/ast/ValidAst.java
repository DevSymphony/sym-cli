/**
 * Test file with valid AST structure
 * Demonstrates proper exception handling and code structure
 */
package com.example;

import java.io.File;
import java.io.FileReader;
import java.io.IOException;
import java.util.logging.Logger;

public class ValidAst {

    private static final Logger LOGGER = Logger.getLogger(ValidAst.class.getName());

    /**
     * Reads file content safely with proper exception handling
     *
     * @param path the file path to read
     * @return the file content
     * @throws IOException if file cannot be read
     */
    public String readFileSafe(String path) throws IOException {
        try (FileReader reader = new FileReader(path)) {
            return "content";
        } catch (IOException e) {
            LOGGER.severe("Failed to read file: " + path);
            throw e;
        }
    }

    /**
     * Performs calculation with proper error handling
     *
     * @param a first operand
     * @param b second operand
     * @param c third operand
     * @return calculated result
     */
    public int calculate(int a, int b, int c) {
        return a + b * c;
    }

    /**
     * Processes data with specific exception handling
     */
    public void processWithSpecificCatch() {
        try {
            riskyOperation();
        } catch (IOException e) {
            LOGGER.warning("I/O error during processing: " + e.getMessage());
            handleError(e);
        }
    }

    private void riskyOperation() throws IOException {
        File file = new File("test.txt");
        if (!file.exists()) {
            throw new IOException("File not found");
        }
    }

    private void handleError(Exception e) {
        LOGGER.severe("Error handled: " + e.getMessage());
    }

    /**
     * Main entry point for the application.
     * @param args command line arguments
     */
    public static void main(String[] args) {
        ValidAst obj = new ValidAst();
        obj.processWithSpecificCatch();
        int result = obj.calculate(1, 2, 3);
        LOGGER.info("Result: " + result);
    }
}
