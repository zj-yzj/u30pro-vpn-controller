package com.codex.u30vpn;

/** Gesture intent only; Android views own scrolling and action rendering. */
final class SwipeGesture {
    enum Action { NONE, TAP, REVEAL, CLOSE, SCROLL }
    private static final int IDLE = 0, UNDECIDED = 1, HORIZONTAL = 2, VERTICAL = 3;
    private final float threshold;
    private final float commitDistance;
    private float startX, startY;
    private int state;

    SwipeGesture(float threshold, float commitDistance) {
        if (threshold <= 0 || commitDistance < threshold) throw new IllegalArgumentException("invalid distances");
        this.threshold = threshold;
        this.commitDistance = commitDistance;
    }

    void down(float x, float y) {
        startX = x;
        startY = y;
        state = UNDECIDED;
    }

    void move(float x, float y) {
        if (state != UNDECIDED) return;
        float horizontal = Math.abs(x - startX), vertical = Math.abs(y - startY);
        if (horizontal > threshold && horizontal > vertical * 1.35f) state = HORIZONTAL;
        else if (vertical > threshold) state = VERTICAL;
    }

    boolean horizontal() { return state == HORIZONTAL; }
    boolean vertical() { return state == VERTICAL; }

    Action up(float x, float y) {
        move(x, y);
        int completed = state;
        state = IDLE;
        if (completed == UNDECIDED) return Action.TAP;
        if (completed == VERTICAL) return Action.SCROLL;
        if (completed != HORIZONTAL) return Action.NONE;
        float distance = x - startX;
        if (distance <= -commitDistance) return Action.REVEAL;
        if (distance >= commitDistance) return Action.CLOSE;
        return Action.NONE;
    }

    void cancel() { state = IDLE; }
}
