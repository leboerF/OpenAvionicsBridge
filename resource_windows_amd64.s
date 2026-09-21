.section .rsrc,"dr"
.balign 4
rsrc_start:
  .long 0, 0
  .short 0, 0, 0, 3
  .long 3
  .long 0x80000000 + icon_type_dir - rsrc_start
  .long 14
  .long 0x80000000 + group_type_dir - rsrc_start
  .long 16
  .long 0x80000000 + version_type_dir - rsrc_start
icon_type_dir:
  .long 0, 0
  .short 0, 0, 0, 7
  .long 1
  .long 0x80000000 + icon_lang_1 - rsrc_start
  .long 2
  .long 0x80000000 + icon_lang_2 - rsrc_start
  .long 3
  .long 0x80000000 + icon_lang_3 - rsrc_start
  .long 4
  .long 0x80000000 + icon_lang_4 - rsrc_start
  .long 5
  .long 0x80000000 + icon_lang_5 - rsrc_start
  .long 6
  .long 0x80000000 + icon_lang_6 - rsrc_start
  .long 7
  .long 0x80000000 + icon_lang_7 - rsrc_start
group_type_dir:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 1
  .long 0x80000000 + group_lang_1 - rsrc_start
version_type_dir:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 1
  .long 0x80000000 + version_lang_1 - rsrc_start
icon_lang_1:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_1 - rsrc_start
icon_lang_2:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_2 - rsrc_start
icon_lang_3:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_3 - rsrc_start
icon_lang_4:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_4 - rsrc_start
icon_lang_5:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_5 - rsrc_start
icon_lang_6:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_6 - rsrc_start
icon_lang_7:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long icon_data_entry_7 - rsrc_start
group_lang_1:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long group_data_entry_1 - rsrc_start
version_lang_1:
  .long 0, 0
  .short 0, 0, 0, 1
  .long 0x0409
  .long version_data_entry_1 - rsrc_start
icon_data_entry_1:
  .rva icon_blob_1
  .long 7848, 0, 0
icon_data_entry_2:
  .rva icon_blob_2
  .long 67624, 0, 0
icon_data_entry_3:
  .rva icon_blob_3
  .long 16936, 0, 0
icon_data_entry_4:
  .rva icon_blob_4
  .long 9640, 0, 0
icon_data_entry_5:
  .rva icon_blob_5
  .long 4264, 0, 0
icon_data_entry_6:
  .rva icon_blob_6
  .long 2440, 0, 0
icon_data_entry_7:
  .rva icon_blob_7
  .long 1128, 0, 0
group_data_entry_1:
  .rva group_blob_1
  .long 104, 0, 0
version_data_entry_1:
  .rva version_blob_1
  .long 868, 0, 0
  .balign 4
icon_blob_1:
  .incbin ".generated-resources/icon_1.bin"
  .balign 4
icon_blob_2:
  .incbin ".generated-resources/icon_2.bin"
  .balign 4
icon_blob_3:
  .incbin ".generated-resources/icon_3.bin"
  .balign 4
icon_blob_4:
  .incbin ".generated-resources/icon_4.bin"
  .balign 4
icon_blob_5:
  .incbin ".generated-resources/icon_5.bin"
  .balign 4
icon_blob_6:
  .incbin ".generated-resources/icon_6.bin"
  .balign 4
icon_blob_7:
  .incbin ".generated-resources/icon_7.bin"
  .balign 4
group_blob_1:
  .short 0, 1, 7
  .byte 0, 0, 0, 0
  .short 1, 32
  .long 7848
  .short 1
  .byte 128, 128, 0, 0
  .short 1, 32
  .long 67624
  .short 2
  .byte 64, 64, 0, 0
  .short 1, 32
  .long 16936
  .short 3
  .byte 48, 48, 0, 0
  .short 1, 32
  .long 9640
  .short 4
  .byte 32, 32, 0, 0
  .short 1, 32
  .long 4264
  .short 5
  .byte 24, 24, 0, 0
  .short 1, 32
  .long 2440
  .short 6
  .byte 16, 16, 0, 0
  .short 1, 32
  .long 1128
  .short 7
  .balign 4
version_blob_1:
  .incbin ".generated-resources/version_info.bin"
  .balign 4
