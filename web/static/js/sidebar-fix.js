// Sidebar management for mobile responsiveness
document.addEventListener('DOMContentLoaded', function() {
  console.log("Sidebar manager loaded");
  
  // Function to ensure proper sidebar behavior on mobile
  function manageSidebar() {
    // Close sidebar when navigation links are clicked on mobile
    const navLinks = document.querySelectorAll('.nav-link');
    const sidebarComponent = document.querySelector('[x-data*="sidebarOpen"]');
    
    navLinks.forEach(link => {
      if (!link.hasAttribute('data-sidebar-handler')) {
        link.setAttribute('data-sidebar-handler', 'true');
        link.addEventListener('click', function() {
          // Only trigger for mobile view
          if (window.innerWidth < 1024) { // lg breakpoint
            // Close sidebar using Alpine.js
            if (typeof Alpine !== 'undefined' && sidebarComponent) {
              Alpine.evaluate(sidebarComponent, 'sidebarOpen = false');
            }
          }
        });
      }
    });
    
    // Remove any duplicate mobile navigation elements that might be injected by JavaScript
    const duplicateNavs = document.querySelectorAll('.mobile-duplicate-nav, .bottom-hamburger, .footer-nav');
    duplicateNavs.forEach(nav => {
      nav.remove();
    });
    
    // Check if we have duplicate sidebars (this shouldn't happen with our updates)
    const sidebarContainers = document.querySelectorAll('.lg\\:fixed.lg\\:inset-y-0.lg\\:z-50.lg\\:flex.lg\\:w-72.lg\\:flex-col');
    if (sidebarContainers.length > 1) {
      console.log('Found ' + sidebarContainers.length + ' sidebars, keeping only the first one');
      
      // Keep only the first sidebar
      for (let i = 1; i < sidebarContainers.length; i++) {
        console.log('Removing duplicate sidebar #' + (i+1));
        sidebarContainers[i].remove();
      }
    }
  }
  
  // Run immediately and after each page update
  manageSidebar();
  
  // Handle HTMX page updates
  document.body.addEventListener('htmx:afterOnLoad', manageSidebar);
  document.body.addEventListener('htmx:afterSwap', manageSidebar);
  document.body.addEventListener('htmx:afterRequest', manageSidebar);
  
  // Also run on window resize
  window.addEventListener('resize', manageSidebar);
  
  // Ensure any HTMX navigation closes the sidebar
  document.addEventListener('htmx:beforeRequest', function(event) {
    const sidebarComponent = document.querySelector('[x-data*="sidebarOpen"]');
    if (typeof Alpine !== 'undefined' && sidebarComponent) {
      Alpine.evaluate(sidebarComponent, 'sidebarOpen = false');
    }
  });
});