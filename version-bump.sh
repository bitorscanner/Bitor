#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to get the latest tag
get_latest_tag() {
    local latest_tag=$(git tag --sort=-version:refname | head -n1)
    if [ -z "$latest_tag" ]; then
        echo "v0.0.0"
    else
        echo "$latest_tag"
    fi
}

# Function to parse version number
parse_version() {
    local version=$1
    # Remove 'v' prefix if present
    version=${version#v}
    
    # Split version into major.minor.patch
    IFS='.' read -r major minor patch <<< "$version"
    
    # Default values if not set
    major=${major:-0}
    minor=${minor:-0}
    patch=${patch:-0}
    
    echo "$major $minor $patch"
}

# Function to increment version
increment_version() {
    local current_version=$1
    local bump_type=$2
    
    read -r major minor patch <<< "$(parse_version "$current_version")"
    
    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            print_error "Invalid bump type: $bump_type"
            exit 1
            ;;
    esac
    
    echo "v$major.$minor.$patch"
}

# Function to show current status
show_status() {
    echo
    print_info "=== Version Management Status ==="
    echo
    print_info "Current branch: $(git rev-parse --abbrev-ref HEAD)"
    print_info "Latest tag: $(get_latest_tag)"
    print_info "Repository: $(git remote get-url origin 2>/dev/null || echo 'No remote configured')"
    echo
    
    # Check if there are uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        print_warning "You have uncommitted changes!"
        git status --porcelain
        echo
    fi
}

# Function to create and push tag
create_and_push_tag() {
    local new_tag=$1
    local message=$2
    
    print_info "Creating tag: $new_tag"
    
    # Create the tag
    if [ -n "$message" ]; then
        git tag -a "$new_tag" -m "$message"
    else
        git tag -a "$new_tag" -m "Release $new_tag"
    fi
    
    print_success "Tag $new_tag created successfully"
    
    # Push the tag
    print_info "Pushing tag to remote..."
    git push origin "$new_tag"
    print_success "Tag $new_tag pushed to remote"
}

# Main function
main() {
    echo
    print_info "🚀 Bitor Version Management Script"
    
    # Show current status
    show_status
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Get current tag
    current_tag=$(get_latest_tag)
    
    # Parse command line arguments
    bump_type=""
    auto_confirm=""
    tag_message=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                bump_type="$2"
                shift 2
                ;;
            -y|--yes)
                auto_confirm="yes"
                shift
                ;;
            -m|--message)
                tag_message="$2"
                shift 2
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo
                echo "Options:"
                echo "  -t, --type TYPE     Version bump type (major, minor, patch)"
                echo "  -y, --yes          Auto-confirm without prompting"
                echo "  -m, --message MSG  Custom tag message"
                echo "  -h, --help         Show this help message"
                echo
                echo "Examples:"
                echo "  $0                           # Interactive mode"
                echo "  $0 -t patch                 # Bump patch version"
                echo "  $0 -t minor -y              # Bump minor, auto-confirm"
                echo "  $0 -t major -m 'Major release with breaking changes'"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # If no bump type specified, ask user
    if [ -z "$bump_type" ]; then
        echo "Current version: $current_tag"
        echo
        echo "Select version bump type:"
        echo "1) patch (${current_tag} → $(increment_version "$current_tag" "patch"))"
        echo "2) minor (${current_tag} → $(increment_version "$current_tag" "minor"))"
        echo "3) major (${current_tag} → $(increment_version "$current_tag" "major"))"
        echo "4) custom"
        echo "5) exit"
        
        read -p "Enter choice (1-5): " choice
        
        case $choice in
            1) bump_type="patch" ;;
            2) bump_type="minor" ;;
            3) bump_type="major" ;;
            4) 
                read -p "Enter custom version (e.g., v1.2.3): " custom_version
                if [[ ! $custom_version =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                    print_error "Invalid version format. Use semantic versioning (e.g., v1.2.3)"
                    exit 1
                fi
                # Add 'v' prefix if not present
                if [[ ! $custom_version =~ ^v ]]; then
                    custom_version="v$custom_version"
                fi
                new_tag="$custom_version"
                ;;
            5) exit 0 ;;
            *) 
                print_error "Invalid choice"
                exit 1
                ;;
        esac
    fi
    
    # Calculate new version (if not custom)
    if [ -z "$new_tag" ]; then
        new_tag=$(increment_version "$current_tag" "$bump_type")
    fi
    
    # Check if tag already exists
    if git rev-parse "$new_tag" >/dev/null 2>&1; then
        print_error "Tag $new_tag already exists"
        exit 1
    fi
    
    # Confirm action
    if [ "$auto_confirm" != "yes" ]; then
        echo
        print_warning "This will:"
        echo "  • Create tag: $new_tag"
        echo "  • Push tag to remote repository"
        echo
        read -p "Continue? (y/N): " confirm
        
        if [[ ! $confirm =~ ^[Yy]$ ]]; then
            print_info "Operation cancelled"
            exit 0
        fi
    fi
    
    # Create and push the tag
    create_and_push_tag "$new_tag" "$tag_message"
    
    echo
    print_success "🎉 Version $new_tag created and pushed successfully!"
    
    echo
    print_info "Next steps:"
    echo "  • Check the GitHub repository for the new tag"
    echo "  • Review GitHub Actions/CI builds if configured"
    echo "  • Update documentation if needed"
    echo "  • Create a release on GitHub if desired"
}

# Run main function with all arguments
main "$@"