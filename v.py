input_file = 's.sql'
output_file = 'q.sql'

with open(input_file, 'r') as infile, open(output_file, 'w') as outfile:
    lines = infile.readlines()

    for line in lines:
        if 'INSERT INTO sub_category' in line:
            parts = line.strip().split(',')

            if len(parts) >= 1:
                # parts[0] = "INSERT INTO sub_category VALUES (123"
                prefix = 'INSERT INTO sub_category VALUES ('
                if parts[0].startswith(prefix):
                    try:
                        id_part = parts[0][len(prefix):].strip()
                        new_id = int(id_part) + 14887
                        parts[0] = f"{prefix}{new_id}"
                        updated_line = ','.join(parts) + '\n'
                        outfile.write(updated_line)
                    except ValueError:
                        # id raqam emas bo‘lsa, asl satrni yozamiz
                        outfile.write(line)
                else:
                    outfile.write(line)
        else:
            outfile.write(line)
