#!/usr/bin/env python3
"""
Document Length Analysis Script

Analyzes markdown documentation files for length issues and provides optimization recommendations.
"""

import os
import re
from collections import defaultdict


def analyze_document_lengths(docs_dir='docs'):
    """Analyze all markdown files in the docs directory"""

    # Document length categories
    length_categories = {
        'short': (1, 100),
        'medium': (101, 300),
        'long': (301, 500),
        'very_long': (501, float('inf'))
    }

    # Optimization recommendations
    optimization_recommendations = {
        'short': 'No optimization needed',
        'medium': 'Light optimization recommended',
        'long': 'Moderate optimization recommended',
        'very_long': 'Significant optimization needed'
    }

    # Analyze each document
    results = []
    category_counts = defaultdict(int)
    total_lines = 0
    total_files = 0

    print("📊 Document Length Analysis")
    print("=" * 50)
    print()

    if not os.path.exists(docs_dir):
        print(f"Directory '{docs_dir}' not found")
        return results

    for filename in os.listdir(docs_dir):
        if filename.endswith('.md'):
            filepath = os.path.join(docs_dir, filename)
            total_files += 1

            with open(filepath, 'r', encoding='utf-8') as f:
                lines = f.readlines()
                line_count = len(lines)
                total_lines += line_count

            # Determine category
            category = 'short'
            for cat, (min_lines, max_lines) in length_categories.items():
                if min_lines <= line_count <= max_lines:
                    category = cat
                    category_counts[cat] += 1
                    break

            # Calculate optimization priority
            if line_count > 500:
                priority = "🔴 HIGH"
            elif line_count > 400:
                priority = "🟡 MEDIUM"
            elif line_count > 300:
                priority = "🟢 LOW"
            else:
                priority = "✅ NONE"

            results.append({
                'filename': filename,
                'line_count': line_count,
                'category': category,
                'recommendation': optimization_recommendations[category],
                'priority': priority
            })

            # Print individual file analysis
            print(f"{filename}: {line_count} lines")
            print(f"  Category: {category.upper().replace('_', ' ')}")
            print(f"  Priority: {priority}")
            print(f"  Recommendation: {optimization_recommendations[category]}")
            print()

    if total_files == 0:
        print("No markdown files found")
        return results

    # Print summary statistics
    print("📈 Summary Statistics")
    print("-" * 30)
    print(f"Total files analyzed: {total_files}")
    print(f"Total lines: {total_lines}")
    print(f"Average lines per file: {total_lines / total_files:.1f}")
    print()

    print("📊 Category Distribution")
    print("-" * 30)
    for category, count in sorted(category_counts.items()):
        percentage = (count / total_files) * 100
        print(f"{category.upper().replace('_', ' ')}: {count} files ({percentage:.1f}%)")
    print()

    # Identify optimization candidates
    high_priority = [r for r in results if r['priority'] in ["🔴 HIGH", "🟡 MEDIUM"]]
    print(f"🎯 Optimization Candidates: {len(high_priority)} files")
    print("-" * 40)
    for result in sorted(high_priority, key=lambda x: x['line_count'], reverse=True):
        print(f"{result['filename']}: {result['line_count']} lines ({result['priority']})")

    return results


if __name__ == "__main__":
    results = analyze_document_lengths()
    print()
    print("🎉 Analysis complete!")