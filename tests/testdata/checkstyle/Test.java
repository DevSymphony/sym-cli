// Checkstyle 위반: TypeName (클래스명 snake_case)
public class bad_class {
    // Checkstyle 위반: MemberName (private 변수 m_ 미사용)
    private int MyVar = 1;

    // Checkstyle 위반: MethodName (메서드명 PascalCase)
    public void BadFunc() {
        final int CONST = 2;
        int result = 10 / CONST;
        System.out.println(result);
    }

    public int getMyVar() {
        return MyVar;
    }
}
