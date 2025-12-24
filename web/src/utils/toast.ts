// Cyberpunk-styled toast utility with app theme colors
export function showToast(
  message: string,
  type: "success" | "error" | "warning" | "info" = "info"
) {
  // Create toast container if it doesn't exist
  let toastContainer = document.getElementById("toast-container");
  if (!toastContainer) {
    toastContainer = document.createElement("div");
    toastContainer.id = "toast-container";
    // Very high z-index to ensure it's above everything including HUD overlay
    toastContainer.style.cssText = `
      position: fixed;
      top: 20px;
      left: 50%;
      transform: translateX(-50%);
      z-index: 9999;
      pointer-events: none;
    `;
    document.body.appendChild(toastContainer);
  }

  // Create toast element with Cyberpunk styling
  const toast = document.createElement("div");

  // Use app theme color (#ABF600 - neon green) for all types
  const themeColor = "#ABF600";
  const themeGlow = "rgba(171, 246, 0, 0.5)";
  const themeBg = "rgba(171, 246, 0, 0.1)";

  toast.style.cssText = `
    position: relative;
    padding: 12px 24px;
    margin-bottom: 10px;
    background: linear-gradient(135deg, ${themeBg}, rgba(0, 0, 0, 0.8));
    border: 1px solid ${themeColor};
    border-radius: 4px;
    box-shadow: 0 0 20px ${themeGlow}, inset 0 0 20px rgba(0, 0, 0, 0.5);
    color: ${themeColor};
    font-family: 'Courier New', monospace;
    font-size: 14px;
    font-weight: bold;
    text-transform: uppercase;
    letter-spacing: 2px;
    backdrop-filter: blur(10px);
    pointer-events: auto;
    animation: slideInDown 0.3s ease-out;
  `;

  // Add message
  const messageSpan = document.createElement("span");
  messageSpan.textContent = message;
  messageSpan.style.cssText = `
    position: relative;
    z-index: 1;
    text-shadow: 0 0 10px ${themeGlow};
  `;
  toast.appendChild(messageSpan);

  // Add animations to document if not already present
  if (!document.getElementById("toast-animations")) {
    const style = document.createElement("style");
    style.id = "toast-animations";
    style.textContent = `
      @keyframes slideInDown {
        from {
          opacity: 0;
          transform: translateY(-20px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }
      @keyframes fadeOut {
        to {
          opacity: 0;
          transform: translateY(-20px);
        }
      }
    `;
    document.head.appendChild(style);
  }

  // Add to container
  toastContainer.appendChild(toast);

  // Remove after 3 seconds
  setTimeout(() => {
    toast.style.animation = "fadeOut 0.3s ease-out forwards";
    setTimeout(() => {
      toastContainer?.removeChild(toast);
      // Remove container if empty
      if (toastContainer?.children.length === 0) {
        document.body.removeChild(toastContainer);
      }
    }, 300);
  }, 3000);
}
