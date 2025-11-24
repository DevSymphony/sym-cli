/**
 * Test file with AST-level violations
 * Contains structural issues detectable via AST analysis
 */
package com.example;

import java.io.File;
import java.io.FileReader;
import java.io.IOException;

public class AstViolations {

    // VIOLATION: System.out usage in production code
    public void debugPrint(String message) {
        System.out.println("Debug: " + message);
    }

    // VIOLATION: File I/O without try-catch
    public String readFileUnsafe(String path) {
        FileReader reader = new FileReader(path);
        return "content";
    }

    // VIOLATION: Empty catch block
    public void emptyCatch() {
        try {
            riskyOperation();
        } catch (Exception e) {
            // Empty catch - swallows exception
        }
    }

    // VIOLATION: Generic exception catch
    public void catchGeneric() {
        try {
            riskyOperation();
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    // VIOLATION: Missing method documentation
    public int calculate(int a, int b, int c) {
        return a + b * c;
    }

    private void riskyOperation() throws IOException {
        File file = new File("test.txt");
        if (!file.exists()) {
            throw new IOException("File not found");
        }
    }

    public static void main(String[] args) {
        AstViolations obj = new AstViolations();
        obj.debugPrint("test");
        obj.emptyCatch();
    }
}
