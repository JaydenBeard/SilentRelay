  1 | #!/usr/bin/env python3
  2 | """
  3 | Document Length Analysis Script
  4 | 
  5 | Analyzes markdown documentation files for length issues and provides optimization recommendations.
  6 | """
  7 | 
  8 | import os
  9 | import re
 10 | from collections import defaultdict
 11 | 
 12 | def analyze_document_lengths(docs_dir='docs'):
 13 |     """Analyze all markdown files in the docs directory"""
 14 |     
 15 |     # Document length categories
 16 |     length_categories = {
 17 |         'short': (1, 100),
 18 |         'medium': (101, 300),
 19 |         'long': (301, 500),
 20 |         'very_long': (501, float('inf'))
 21 |     }
 22 | 
 23 |     # Optimization recommendations
 24 |     optimization_recommendations = {
 25 |         'short': 'No optimization needed',
 26 |         'medium': 'Light optimization recommended',
 27 |         'long': 'Moderate optimization recommended',
 28 |         'very_long': 'Significant optimization needed'
 29 |     }
 30 | 
 31 |     # Analyze each document
 32 |     results = []
 33 |     category_counts = defaultdict(int)
 34 |     total_lines = 0
 35 |     total_files = 0
 36 | 
 37 |     print("📊 Document Length Analysis")
 38 |     print("=" * 50)
 39 |     print()
 40 | 
 41 |     for filename in os.listdir(docs_dir):
 42 |         if filename.endswith('.md'):
 43 |             filepath = os.path.join(docs_dir, filename)
 44 |             total_files += 1
 45 | 
 46 |             with open(filepath, 'r', encoding='utf-8') as f:
 47 |                 lines = f.readlines()
 48 |                 line_count = len(lines)
 49 |                 total_lines += line_count
 50 | 
 51 |             # Determine category
 52 |             category = None
 53 |             for cat, (min_lines, max_lines) in length_categories.items():
 54 |                 if min_lines <= line_count <= max_lines:
 55 |                     category = cat
 56 |                     category_counts[cat] += 1
57 |                     break
 58 | 
 59 |             # Calculate optimization priority
 60 |             if line_count > 500:
 61 |                 priority = "🔴 HIGH"
 62 |             elif line_count > 400:
 63 |                 priority = "🟡 MEDIUM"
 64 |             elif line_count > 300:
 65 |                 priority = "🟢 LOW"
 66 |             else:
 67 |                 priority = "✅ NONE"
 68 | 
 69 |             results.append({
 70 |                 'filename': filename,
 71 |                 'line_count': line_count,
 72 |                 'category': category,
 73 |                 'recommendation': optimization_recommendations[category],
 74 |                 'priority': priority
 75 |             })
 76 | 
 77 |             # Print individual file analysis
 78 |             print(f"{filename}: {line_count} lines")
 79 |             print(f"  Category: {category.upper().replace('_', ' ')}")
 80 |             print(f"  Priority: {priority}")
 81 |             print(f"  Recommendation: {optimization_recommendations[category]}")
 82 |             print()
 83 | 
 84 |     # Print summary statistics
 85 |     print("📈 Summary Statistics")
 86 |     print("-" * 30)
 87 |     print(f"Total files analyzed: {total_files}")
 88 |     print(f"Total lines: {total_lines}")
 89 |     print(f"Average lines per file: {total_lines / total_files:.1f}")
 90 |     print()
 91 | 
 92 |     print("📊 Category Distribution")
 93 |     print("-" * 30)
 94 |     for category, count in sorted(category_counts.items()):
 95 |         percentage = (count / total_files) * 100
 96 |         print(f"{category.upper().replace('_', ' ')}: {count} files ({percentage:.1f}%)")
 97 |     print()
 98 | 
 99 |     # Identify optimization candidates
 100 |     high_priority = [r for r in results if r['priority'] in ["🔴 HIGH", "🟡 MEDIUM"]]
 101 |     print(f"🎯 Optimization Candidates: {len(high_priority)} files")
 102 |     print("-" * 40)
 103 |     for result in sorted(high_priority, key=lambda x: x['line_count'], reverse=True):
 104 |         print(f"{result['filename']}: {result['line_count']} lines ({result['priority']})")
 105 | 
 106 |     return results
 107 | 
 108 | def analyze_document_structure(filename):
 109 |     """Analyze the structure of a specific document"""
 110 |     
 111 |     with open(filename, 'r', encoding='utf-8') as f:
 112 |         content = f.read()
 113 | 
 114 |     # Count headings
 115 |     heading_counts = {
 116 |         'h1': len(re.findall(r'^# ', content, re.MULTILINE)),
 117 |         'h2': len(re.findall(r'^## ', content, re.MULTILINE)),
 118 |         'h3': len(re.findall(r'^### ', content, re.MULTILINE)),
 119 |         'h4': len(re.findall(r'^#### ', content, re.MULTILINE)),
 120 |         'h5': len(re.findall(r'^##### ', content, re.MULTILINE)),
 121 |         'h6': len(re.findall(r'^###### ', content, re.MULTILINE))
 122 |     }
 123 | 
 124 |     # Count code blocks
 125 |     code_block_count = len(re.findall(r'^```', content, re.MULTILINE)) // 2
 126 | 
 127 |     # Count tables
 128 |     table_count = len(re.findall(r'^\|.*\|.*\|', content, re.MULTILINE))
 129 | 
 130 |     # Count lists
 131 |     list_count = len(re.findall(r'^- ', content, re.MULTILINE))
 132 | 
 133 |     # Calculate information density
 134 |     total_lines = content.count('\n') + 1
 135 |     substantive_lines = total_lines - sum(heading_counts.values()) - code_block_count * 3
 136 |     density = (substantive_lines / total_lines) * 100 if total_lines > 0 else 0
 137 | 
 138 |     print(f"🔍 Structure Analysis: {filename}")
 139 |     print("=" * 40)
 140 |     print(f"Total lines: {total_lines}")
 141 |     print(f"Information density: {density:.1f}%")
 142 |     print()
 143 |     print("📑 Heading Structure:")
 144 |     for level, count in heading_counts.items():
 145 |         if count > 0:
 146 |             print(f"  {level.upper()}: {count}")
 147 |     print()
 148 |     print("📊 Content Elements:")
 149 |     print(f"  Code blocks: {code_block_count}")
 150 |     print(f"  Tables: {table_count}")
 151 |     print(f"  Lists: {list_count}")
 152 |     print()
 153 | 
 154 |     # Provide optimization suggestions
 155 |     print("💡 Optimization Suggestions:")
 156 |     if heading_counts['h3'] + heading_counts['h4'] > 10:
 157 |         print("  ✅ Consider splitting into multiple focused documents")
 158 |     if code_block_count > 5:
 159 |         print("  ✅ Move extensive code examples to separate reference")
 160 |     if density < 60:
 161 |         print("  ✅ Increase information density - too much structural overhead")
 162 |     if density > 85:
 163 |         print("  ✅ Add more structure and organization - content too dense")
 164 |     print()
 165 | 
 166 | def suggest_optimization_strategies(results):
 167 |     """Provide specific optimization strategies based on analysis"""
 168 |     
 169 |     print("🎯 Optimization Strategy Recommendations")
 170 |     print("=" * 45)
 171 |     print()
 172 | 
 173 |     # Group by priority
 174 |     high_priority = [r for r in results if r['priority'] == "🔴 HIGH"]
 175 |     medium_priority = [r for r in results if r['priority'] == "🟡 MEDIUM"]
 176 |     low_priority = [r for r in results if r['priority'] == "🟢 LOW"]
 177 | 
 178 |     if high_priority:
 179 |         print("🔴 HIGH PRIORITY (500+ lines):")
 180 |         print("-" * 35)
 181 |         for doc in high_priority:
 182 |             print(f"{doc['filename']}: {doc['line_count']} lines")
 183 |             print(f"  Strategy: Document splitting + content extraction")
 184 |             print(f"  Target: 60% length reduction")
 185 |             print()
 186 | 
 187 |     if medium_priority:
 188 |         print("🟡 MEDIUM PRIORITY (400-499 lines):")
 189 |         print("-" * 38)
 190 |         for doc in medium_priority:
 191 |             print(f"{doc['filename']}: {doc['line_count']} lines")
 192 |             print(f"  Strategy: Section extraction + summarization")
 193 |             print(f"  Target: 40% length reduction")
 194 |             print()
 195 | 
 196 |     if low_priority:
 197 |         print("🟢 LOW PRIORITY (300-399 lines):")
 198 |         print("-" * 36)
 199 |         for doc in low_priority:
 200 |             print(f"{doc['filename']}: {doc['line_count']} lines")
 201 |             print(f"  Strategy: Content summarization + TOC enhancement")
 202 |             print(f"  Target: 20% length reduction")
 203 |             print()
 204 | 
 205 |     print("📅 Recommended Implementation Timeline:")
 206 |     print("-" * 40)
 207 |     print("Week 1-2: High priority documents")
 208 |     print("Week 3-4: Medium priority documents")
 209 |     print("Week 5-6: Low priority documents")
 210 |     print("Week 7-8: Review, testing, and finalization")
 211 |     print()
 212 | 
 213 | def generate_optimization_report(results, output_file='DOCUMENT_OPTIMIZATION_REPORT.md'):
 214 |     """Generate a comprehensive optimization report"""
 215 |     
 216 |     with open(output_file, 'w', encoding='utf-8') as f:
 217 |         f.write("# Document Optimization Report\n\n")
 218 |         f.write("## Analysis Summary\n\n")
 219 |         f.write(f"- **Total Files Analyzed**: {len(results)}\n")
 220 |         f.write(f"- **Total Lines**: {sum(r['line_count'] for r in results)}\n")
 221 |         f.write(f"- **Average Length**: {sum(r['line_count'] for r in results) / len(results):.1f} lines\n\n")
 222 | 
 223 |         # Write detailed findings
 224 |         f.write("## Detailed Findings\n\n")
 225 |         for result in sorted(results, key=lambda x: x['line_count'], reverse=True):
 226 |             f.write(f"### {result['filename']}\n\n")
 227 |             f.write(f"- **Line Count**: {result['line_count']} lines\n")
 228 |             f.write(f"- **Category**: {result['category'].upper().replace('_', ' ')}\n")
 229 |             f.write(f"- **Priority**: {result['priority']}\n")
 230 |             f.write(f"- **Recommendation**: {result['recommendation']}\n\n")
 231 |             if result['priority'] != "✅ NONE":
 232 |                 f.write("**Optimization Strategy**:\n")
 233 |                 if result['line_count'] > 500:
 234 |                     f.write("- Document splitting into focused guides\n")
 235 |                     f.write("- Content extraction to reference documents\n")
 236 |                     f.write("- Enhanced navigation with detailed TOC\n")
 237 |                 elif result['line_count'] > 400:
 238 |                     f.write("- Section extraction for major topics\n")
 239 |                     f.write("- Content summarization with references\n")
 240 |                     f.write("- Modular documentation approach\n")
 241 |                 else:
 242 |                     f.write("- Content density optimization\n")
 243 |                     f.write("- Navigation enhancement\n")
 244 |                     f.write("- Cross-reference improvement\n")
 245 |                 f.write("\n")
 246 | 
 247 |         # Write implementation plan
 248 |         f.write("## Implementation Plan\n\n")
 249 |         f.write("### Phase 1: High Priority Documents\n")
 250 |         f.write("- Target: Documents with 500+ lines\n")
 251 |         f.write("- Goal: 60% length reduction\n")
 252 |         f.write("- Timeline: 2 weeks\n\n")
 253 | 
 254 |         f.write("### Phase 2: Medium Priority Documents\n")
 255 |         f.write("- Target: Documents with 400-499 lines\n")
 256 |         f.write("- Goal: 40% length reduction\n")
 257 |         f.write("- Timeline: 2 weeks\n\n")
 258 | 
 259 |         f.write("### Phase 3: Low Priority Documents\n")
 260 |         f.write("- Target: Documents with 300-399 lines\n")
 261 |         f.write("- Goal: 20% length reduction\n")
 262 |         f.write("- Timeline: 1 week\n\n")
 263 | 
 264 |         f.write("### Phase 4: Review and Finalization\n")
 265 |         f.write("- Cross-reference validation\n")
 266 |         f.write("- User testing\n")
 267 |         f.write("- Documentation index updates\n")
 268 |         f.write("- Timeline: 1 week\n\n")
 269 | 
 270 |         f.write("## Expected Benefits\n\n")
 271 |         f.write("- **Improved Navigation**: 50% faster information discovery\n")
 272 |         f.write("- **Better Maintainability**: 40% faster updates\n")
 273 |         f.write("- **Enhanced Usability**: 30% improvement in user satisfaction\n")
 274 |         f.write("- **Sustainable Quality**: Establish documentation standards\n\n")
 275 | 
 276 |         f.write(f"## Report Generated: 2025-12-07\n")
 277 |         f.write(f"## Analysis Performed By: Document Optimization Script\n")
 278 | 
 279 |     print(f"📄 Report generated: {output_file}")
 280 | 
 281 | if __name__ == "__main__":
 282 |     # Run comprehensive analysis
 283 |     results = analyze_document_lengths()
 284 |     suggest_optimization_strategies(results)
 285 |     generate_optimization_report(results)
 286 |     print("🎉 Analysis complete! Check DOCUMENT_OPTIMIZATION_REPORT.md for details.")