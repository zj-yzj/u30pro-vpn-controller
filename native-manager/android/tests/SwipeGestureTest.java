package com.codex.u30vpn;

/** Host-only intent arbitration; no Android touch mocks or rendering assertions. */
public final class SwipeGestureTest {
    public static void main(String[] args) {
        SwipeGesture gesture = new SwipeGesture(8, 20);
        gesture.down(50, 30);
        gesture.move(53, 32);
        check(gesture.up(53, 32) == SwipeGesture.Action.TAP, "a stationary touch is a tap");
        gesture.down(70, 30);
        gesture.move(40, 33);
        check(gesture.horizontal(), "dominant left movement claims horizontal gesture");
        check(gesture.up(35, 33) == SwipeGesture.Action.REVEAL, "left swipe reveals action");
        gesture.down(35, 30);
        gesture.move(70, 32);
        check(gesture.up(75, 32) == SwipeGesture.Action.CLOSE, "right swipe closes action");
        gesture.down(50, 30);
        gesture.move(47, 58);
        check(gesture.vertical(), "vertical movement releases parent scrolling");
        check(gesture.up(47, 65) == SwipeGesture.Action.SCROLL, "vertical drag never selects");
        gesture.down(50, 30);
        gesture.move(40, 42);
        check(gesture.vertical(), "ambiguous diagonal drag favors vertical scrolling");
        check(gesture.up(20, 70) == SwipeGesture.Action.SCROLL, "vertical intent stays vertical");
        gesture.down(70, 30);
        gesture.move(40, 31);
        gesture.cancel();
        check(gesture.up(40, 31) == SwipeGesture.Action.NONE, "cancel never clicks or reveals");
        gesture.down(70, 30);
        gesture.move(58, 31);
        check(gesture.up(67, 30) == SwipeGesture.Action.NONE, "short or reversed horizontal drag suppresses click");
        gesture.down(70, 30);
        gesture.move(42, 31);
        gesture.move(41, 60);
        check(gesture.horizontal(), "claimed horizontal intent does not turn into scrolling");
        check(gesture.up(40, 60) == SwipeGesture.Action.REVEAL, "horizontal completion uses final direction");
        gesture.down(50, 30);
        check(gesture.up(50, 30) == SwipeGesture.Action.TAP, "new touch resets prior state");
        System.out.println("SWIPE_GESTURE_TEST_OK: tap, reveal, close, vertical, diagonal, cancel, reversed drag, reset");
    }

    private static void check(boolean actual, String message) {
        if (!actual) throw new AssertionError(message);
    }
}
