import random

def generate_non_overlapping_ranges(start, end, range_length, num_ranges):
    available_space = end - start - range_length + 1
    if available_space < num_ranges * range_length:
        raise ValueError("Not enough space to generate non-overlapping ranges")

    ranges = []
    used_starts = set()

    while len(ranges) < num_ranges:
        range_start = random.randint(start, end - range_length + 1)
        range_end = range_start + range_length - 1

        if range_start not in used_starts and all(
            range_start > r[1] or range_end < r[0] for r in ranges
        ):
            ranges.append((range_start, range_end))
            used_starts.update(range(range_start, range_end + 1))

    return sorted(ranges)

# Set parameters
start = 18908893
end = 20000000
range_length = 100
num_ranges = 20

# Generate ranges
try:
    generated_ranges = generate_non_overlapping_ranges(start, end, range_length, num_ranges)

    # Write ranges to file
    with open('range.txt', 'w') as f:
        for range_start, range_end in generated_ranges:
            f.write(f"{range_start}-{range_end}\n")

    print("Ranges have been successfully generated and written to range.txt")
except ValueError as e:
    print(f"Error: {e}")