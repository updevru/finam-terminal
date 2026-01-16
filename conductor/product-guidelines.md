# Product Guidelines - Finam Terminal TUI

## Documentation & Communication
- **Prose Style:** Technical and concise. Documentation, log messages, and user-facing text should be precise, brief, and focused on providing direct value to developers and power users.
- **Tone:** Professional, direct, and functional. Avoid fluff or overly conversational language.
- **Language:** Primary documentation is in English. High-priority user-facing error messages and specific status indicators may use Russian to ensure immediate clarity for the target audience.

## TUI Design Principles
- **Information Density:** Prioritize the efficient display of data. UI elements should be compact, allowing as much market and portfolio information as possible to be visible without overwhelming the user.
- **Functional Aesthetics:** The visual design must serve the data. Use a high-contrast "Classic Terminal" aesthetic (e.g., green/amber on black) by default, while allowing for modern color highlights to indicate status.
- **No Decoration:** Avoid purely decorative UI elements (like large borders or unnecessary padding) that do not contribute to data clarity or navigation.

## Error Handling & Feedback
- **Actionable UI Messages:** Errors should be displayed prominently within the TUI using clear, color-coded indicators.
- **Clarity over Detail:** Provide the user with a concise description of the error and a suggested action (e.g., "Invalid Token - Check .env") rather than raw technical stack traces in the primary UI.
- **Background Logging:** Full technical details for all errors should be captured in background logs for deeper debugging without cluttering the user experience.

## User Experience & Navigation
- **Keyboard-First:** All navigation and actions must be accessible via standard terminal keyboard shortcuts (Arrows, Enter, Esc) and specific hotkeys defined in a global help menu.
- **Efficiency:** Design for rapid data context switching. A user should be able to move between symbols or account views with minimal keystrokes.

## Real-Time Data Representation
- **Dynamic Highlighting:** Implement "Flash on Change" logic for critical metrics like price and size. Briefly highlight values in green (increase) or red (decrease) to provide immediate visual feedback on market movements.
- **Persistent Layouts:** Maintain a stable UI layout where data updates in-place, ensuring the user's focus isn't disrupted by shifting interface elements.
