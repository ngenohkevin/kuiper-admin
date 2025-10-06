// main.js - Enhanced functionality for Ganymede Admin

document.addEventListener('DOMContentLoaded', function() {
  console.log("Main.js loaded");
  
  // Fix for duplicate sidebars
  // - REMOVED implementation to fix the sidebar issue -
  
  // Setup toast notifications
  window.showToast = function(message, type = 'info', duration = 3000) {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type} fade-in`;
    toast.innerHTML = `
      <div class="flex items-center">
        <div class="flex-shrink-0">
          ${getToastIcon(type)}
        </div>
        <div class="ml-3">
          <p class="text-sm font-medium">${message}</p>
        </div>
        <div class="ml-auto pl-3">
          <div class="-mx-1.5 -my-1.5">
            <button type="button" class="inline-flex rounded-md p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 focus:outline-none">
              <span class="sr-only">Dismiss</span>
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    `;
    
    document.body.appendChild(toast);
    
    // Add click event to dismiss button
    const closeButton = toast.querySelector('button');
    closeButton.addEventListener('click', function() {
      toast.classList.add('opacity-0');
      setTimeout(() => {
        toast.remove();
      }, 300);
    });
    
    // Auto-dismiss after duration
    setTimeout(() => {
      toast.classList.add('opacity-0');
      setTimeout(() => {
        toast.remove();
      }, 300);
    }, duration);
  };
  
  // Helper function to get the appropriate icon for toast type
  function getToastIcon(type) {
    switch(type) {
      case 'success':
        return `<svg class="h-5 w-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>`;
      case 'error':
        return `<svg class="h-5 w-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        </svg>`;
      case 'info':
      default:
        return `<svg class="h-5 w-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
        </svg>`;
    }
  }
  
  // Set up active navigation highlighting
  function setupActiveNavigation() {
    // Get current path
    const currentPath = window.location.pathname;
    
    // Find all navigation links
    const navLinks = document.querySelectorAll('.nav-link');
    
    // Clear any existing active classes
    navLinks.forEach(link => {
      // If the link's href matches the current path, add 'active' class
      const href = link.getAttribute('href');
      if (href === currentPath || 
          (currentPath.startsWith('/categories') && href === '/categories') ||
          (currentPath.startsWith('/products') && href === '/products') ||
          (currentPath.startsWith('/reviews') && href === '/reviews') ||
          (currentPath === '/' && href === '/')) {
        link.classList.add('active');
        // Also set the icon color
        const icon = link.querySelector('svg');
        if (icon) {
          icon.classList.add('text-primary');
          icon.classList.remove('text-gray-500', 'dark:text-gray-400');
        }
      } else {
        link.classList.remove('active');
        // Reset icon color
        const icon = link.querySelector('svg');
        if (icon) {
          icon.classList.remove('text-primary');
          icon.classList.add('text-gray-500', 'dark:text-gray-400');
        }
      }
    });
  }
  
  // Initialize active navigation
  setupActiveNavigation();
  
  // Update active navigation after page load
  document.body.addEventListener('htmx:afterSwap', setupActiveNavigation);
});
