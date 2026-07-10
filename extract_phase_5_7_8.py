import os
import sys
from PyPDF2 import PdfReader

root = r'C:\Users\User\Downloads\Skill Arena Game\Documents and planning'

# PDFs to extract
pdfs_to_extract = [
    'phase 5 tech spec and engineer/Phase_5_Master_Overview_Technical_Architecture_Blueprint.pdf',
    'phase 5 tech spec and engineer/part 1/Phase_5_Document_1_System_Backend_Microservices_Architecture.pdf',
    'Phase 7 maze generation and anti bot/Phase_7_Master_Overview_Challenge_Intelligence_Blueprint.pdf',
    'Phase 7 maze generation and anti bot/Part 1 The Maze/Phase_7_Document_1_Maze_Generation_Difficulty_House_Probability_Architecture.pdf',
    'Phase 8 Game Flow/Phase_8_Document_8_Maze_Game_Master_Specification_Implementation_Blueprint.pdf',
    'Phase 8 Game Flow/Part 2 Maze Mechanics/Phase_8_Document_2_Maze_Mechanics_Gameplay_Rules_Specification.pdf'
]

output = []

for pdf_rel_path in pdfs_to_extract:
    pdf_path = os.path.join(root, pdf_rel_path)
    output.append(f"\n{'='*100}\n")
    output.append(f"FILE: {pdf_rel_path}\n")
    output.append(f"{'='*100}\n")
    
    if not os.path.exists(pdf_path):
        output.append(f"ERROR: File not found at {pdf_path}\n")
        continue
    
    try:
        reader = PdfReader(pdf_path)
        output.append(f"TOTAL PAGES: {len(reader.pages)}\n\n")
        
        # Extract first 5 pages for overview
        for i, page in enumerate(reader.pages[:5]):
            output.append(f"\n--- PAGE {i+1} ---\n")
            page_text = page.extract_text()
            if page_text:
                output.append(page_text)
            else:
                output.append("[No extractable text on this page]")
        
        output.append(f"\n\n[... Document continues for {len(reader.pages) - 5} more pages ...]\n")
    except Exception as e:
        output.append(f"ERROR: {str(e)}\n")

# Write to file
with open(r'C:\Users\User\Downloads\Skill Arena Game\PHASE_5_7_8_EXTRACTED.txt', 'w', encoding='utf-8') as f:
    f.writelines(output)

print(f"Extraction complete! Output written to PHASE_5_7_8_EXTRACTED.txt")
print(f"Total length: {sum(len(line) for line in output)} characters")
