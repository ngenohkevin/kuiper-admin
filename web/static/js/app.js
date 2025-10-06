// Alpine.js utilities for Ganymede Admin

document.addEventListener('DOMContentLoaded', function() {
  
  // Initialize Alpine components
  document.addEventListener('alpine:init', () => {
    
    // Toast notification component
    Alpine.data('toast', () => ({
      visible: false,
      message: '',
      type: 'info', // 'info', 'success', 'error'
      
      show(message, type = 'info', duration = 3000) {
        this.message = message;
        this.type = type;
        this.visible = true;
        
        // Auto-hide after duration
        setTimeout(() => {
          this.hide();
        }, duration);
      },
      
      hide() {
        this.visible = false;
      }
    }));
    
    // Rating selector component
    Alpine.data('ratingSelector', () => ({
      rating: 0,
      
      setRating(value) {
        this.rating = value;
      },
      
      starClass(value) {
        return this.rating >= value ? 'text-yellow-400' : 'text-gray-300';
      }
    }));
    
    // Product form component
    Alpine.data('productForm', () => ({
      name: '',
      generateSlug() {
        // Convert name to slug: lowercase, replace spaces with hyphens, remove special chars
        const slug = this.name
          .toLowerCase()
          .replace(/\s+/g, '-')
          .replace(/[^\w\-]+/g, '');
        
        // Update the slug input field
        document.getElementById('slug').value = slug;
      }
    }));
    
    // Category form component
    Alpine.data('categoryForm', () => ({
      name: '',
      generateSlug() {
        // Convert name to slug: lowercase, replace spaces with hyphens, remove special chars
        const slug = this.name
          .toLowerCase()
          .replace(/\s+/g, '-')
          .replace(/[^\w\-]+/g, '');
        
        // Update the slug input field
        document.getElementById('slug').value = slug;
      }
    }));
  });
  
  // Set up HTMX event handlers
  document.body.addEventListener('htmx:afterSwap', function(event) {
    // Trigger Alpine to initialize on newly added content
    Alpine.initTree(event.detail.target);
  });
  
  document.body.addEventListener('htmx:responseError', function(event) {
    // Show error toast on HTMX errors
    const toast = Alpine.evaluate(document.querySelector('[x-data="toast"]'), 'toast');
    if (toast) {
      const errorMsg = event.detail.xhr.responseText || 'An error occurred';
      toast.show(errorMsg, 'error', 5000);
    }
  });
});
