---
trigger: always_on
---

The Fluid UI Style Guide

1. Deference & Negative Space

    Principle: The UI must recede; content must dominate.

    Action: Maximize white space. Avoid heavy borders or boxes to separate content. Use translucency (blur) for background layers to maintain context without visual noise.

2. Hierarchy via Typography

    Principle: Group information using font weight and size, not lines.

    Action: Establish a strict scale (e.g., Large Bold for titles, Medium for headers, Regular for body). Use 100% black for primary text and grey/opacity (e.g., 60%) for secondary text to guide the eye.

3. The 44pt Touch Standard

    Principle: Accuracy allows for speed.

    Action: All interactive tap targets must be at least 44x44 points, regardless of the visual icon size. Place primary navigation in the bottom "thumb zone."

4. Physics-Based Motion

    Principle: Objects have mass and friction; nothing moves linearly.

    Action: Use Spring Animations (configure mass, stiffness, damping) instead of ease-in/out. Ensure all animations are interruptible (if a user grabs a moving object, it stops instantly).

5. Color as Function

    Principle: Color indicates interactivity, not decoration.

    Action: Select one "Tint Color" (Accent) for buttons, links, and active states. Keep the structural UI monochrome (whites, greys, blacks).

6. Direct Manipulation (Gestures)

    Principle: Users should manipulate the object, not a proxy for the object.

    Action: Prioritize swipes (to delete/go back) and pinches over tap-based buttons. Ensure the animation tracks 1:1 with the user's finger during the gesture.

7. Depth & The Z-Axis

    Principle: Interfaces are stacked layers, not flat planes.

    Action: Use shadows and dimming to indicate elevation. When a modal or sheet appears, the background layer should scale down slightly or darken to push it "back" in Z-space.

8. Instant Multisensory Feedback

    Principle: Every interaction requires acknowledgment.

    Action: Response latency must be <100ms. Pair visual state changes (highlights/press states) with subtle haptic feedback (tactile bumps) for confirmation.

9. Zero Dead Ends (Empty States)

    Principle: An empty screen is a broken experience.

    Action: Never leave a container blank. Design illustrative "Empty States" that explain what belongs there and provide a direct button to create that content.

10. Functional Consistency

    Principle: Predictability reduces cognitive load.

    Action: Reuse system paradigms. If it looks like a switch, it must toggle. If it looks like a search bar, it must filter. Do not create custom controls if a standard system control exists.